package upgrade

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

func (sv SemanticVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
}

func (sv SemanticVersion) Next(chg change) (next SemanticVersion) {
	next = sv
	switch chg {
	case breakingChange:
		if next.Major > 0 {
			next.Major++
			next.Minor = 0
			next.Patch = 0
		} else {
			next.Minor++
			next.Patch = 0
		}
	case somethingNew:
		next.Minor++
		next.Patch = 0
	case justPatch:
		next.Patch++
	case noChange:
	}
	return next
}

func parse(s string) (sv SemanticVersion, err error) {
	var (
		v  string
		ok bool
	)
	s = strings.TrimPrefix(s, "v")
	v, s, ok = strings.Cut(s, ".")
	if !ok {
		return SemanticVersion{}, fmt.Errorf("unable to parse major version: malformed version %q", s)
	}
	sv.Major, err = strconv.Atoi(v)
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("unable to parse major version: %w", err)
	}
	v, s, ok = strings.Cut(s, ".")
	if !ok {
		return SemanticVersion{}, fmt.Errorf("unable to parse minor version: malformed version %q", s)
	}
	sv.Minor, err = strconv.Atoi(v)
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("unable to parse minor version: %w", err)
	}
	for i, ch := range s {
		if !('0' <= ch && ch <= '9') {
			v = s[:i]
			break // Don't forget to break out of the loop.
		} else if i == len(s)-1 {
			v = s
		}
	}
	sv.Patch, err = strconv.Atoi(v)
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("unable to parse patch version: %w", err)
	}
	return sv, nil
}

func detectChange() (change, error) {
	files, err := gitChangedFiles()
	if err != nil {
		return noChange, err
	}
	// Divide files in the same directory into a group, because usually .go files in
	// the same directory belong to the same go package.
	var (
		dirFileMap = make(map[string][]string)
	)
	for _, file := range files {
		dir := filepath.Dir(file)
		dirFileMap[dir] = append(dirFileMap[dir], file)
	}
	var topChange change
	for _, files = range dirFileMap {
		var fileChange change
		fileChange, err = diff(files)
		if err != nil {
			return noChange, err
		}
		if fileChange > topChange {
			topChange = fileChange
		}
	}
	return topChange, nil
}

var (
	ErrFileDoesNotExist = errors.New("file does not exist")
)

func gitChangedFiles() ([]string, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	files := make([]string, 0, 8)
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			if file := fields[len(fields)-1]; filepath.Ext(file) == ".go" {
				files = append(files, file)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %q output: %w", "git status", err)
	}
	return files, nil
}

func gitShow(branch string, file string) ([]byte, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.Command("git", "show", branch+":"+file)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), fmt.Sprintf("but not in '%s'", branch)) {
			return nil, fmt.Errorf("%w in %q", ErrFileDoesNotExist, branch)
		}
		return nil, fmt.Errorf("%s: %w", strings.Join(cmd.Args, " "), err)
	}
	return stdout.Bytes(), nil
}

type change int

const (
	noChange change = iota
	justPatch
	somethingNew
	breakingChange
)

func diff(files []string) (change, error) {
	var (
		oldFileSet = token.NewFileSet()
		newFileSet = token.NewFileSet()
		oldAst     *ast.File
		newAst     *ast.File
	)
	// Determine whether there are any additions or deletions to global variables,
	// types, and functions.
	var (
		oldTypeMap = make(map[string]*ast.TypeSpec)
		oldVarMap  = make(map[string]*ast.ValueSpec)
		oldFuncMap = make(map[string]*ast.FuncDecl)

		newTypeMap = make(map[string]*ast.TypeSpec)
		newVarMap  = make(map[string]*ast.ValueSpec)
		newFuncMap = make(map[string]*ast.FuncDecl)
	)
	// Compare all *ast.Decl under the same package uniformly to handle the situation
	// where a *ast.Decl migrates from one file to another.
	for _, file := range files {
		// Attempting to find the "previous" commit record of the currently changed file in the git history,
		// and parse its content into *ast.File (this will not be executed for new files, as new files do
		// not have a git history of commits).
		oldFileSrc, err := gitShow("HEAD", file)
		if err != nil && !errors.Is(err, ErrFileDoesNotExist) {
			return noChange, err
		}
		if oldFileSrc != nil {
			oldAst, err = parser.ParseFile(oldFileSet, file, oldFileSrc, 0)
			if err != nil {
				return noChange, err
			}
			inspectDecls(oldAst, oldTypeMap, oldVarMap, oldFuncMap)
		}
		newAst, err = parser.ParseFile(newFileSet, file, nil, 0)
		if err != nil {
			return noChange, err
		}
		inspectDecls(newAst, newTypeMap, newVarMap, newFuncMap)
	}
	var addition bool
	for name, oldTypeSpec := range oldTypeMap {
		newTypeSpec, ok := newTypeMap[name]
		if !ok {
			return breakingChange, nil
		}
		switch typeChange := typeDiff(oldTypeSpec, newTypeSpec); typeChange {
		case breakingChange:
			return breakingChange, nil
		case somethingNew:
			addition = true
		default:
		}
	}
	for name, oldVarSpec := range oldVarMap {
		newVarSpec, ok := newVarMap[name]
		if !ok {
			return breakingChange, nil
		}
		if oldVarSpec.Type != nil && newVarSpec.Type != nil {
			// Regarding the types in variable definitions, any modification is considered
			// a breaking change.
			if typeExprChange := typeExprDiff(oldVarSpec.Type, newVarSpec.Type); typeExprChange != noChange {
				return breakingChange, nil
			}
		}
		// The definition of constants and variables does not require judging whether
		// their assignment expressions are consistent, because different expressions
		// may produce the same value.
	}
	for name, oldFuncDecl := range oldFuncMap {
		newFuncDecl, ok := newFuncMap[name]
		if !ok {
			return breakingChange, nil
		}
		// In the definition of functions and methods, any changes are considered breaking changes,
		// including changes to the receiver type, type parameters, function parameters, function
		// return value positions (with the exception that changing parameter names is not considered
		// a breaking change), and situations where various types of parameters are added or removed.
		oldFuncType, newFuncType := oldFuncDecl.Type, newFuncDecl.Type
		if fieldsChange := posFieldsDiff(oldFuncType.TypeParams, newFuncType.TypeParams); fieldsChange != noChange {
			return breakingChange, nil
		}
		if fieldsChange := posFieldsDiff(oldFuncType.Params, newFuncType.Params); fieldsChange != noChange {
			return breakingChange, nil
		}
		if fieldsChange := posFieldsDiff(oldFuncType.Results, newFuncType.Results); fieldsChange != noChange {
			return breakingChange, nil
		}
	}
	// At this point, all *ast.Decl in the content of old files have been matched one-to-one with
	// *ast.Decl in the content of new files. If at this time, the number of *ast.Decl in the new
	// file content is more than that in the old files, it indicates that there are additional
	// types, variables, or functions.
	addition = addition ||
		len(newTypeMap) > len(oldTypeMap) ||
		len(newVarMap) > len(oldVarMap) ||
		len(newFuncMap) > len(oldFuncMap)
	if addition {
		return somethingNew, nil
	}
	return justPatch, nil
}

func inspectDecls(
	file *ast.File,
	typeMap map[string]*ast.TypeSpec,
	varMap map[string]*ast.ValueSpec,
	funcMap map[string]*ast.FuncDecl,
) {
	// If a file comes with a build tag, then all *ast.Decl under that file will
	// carry the prefix of the build tag included in that file.
	//
	// This also means that moving a public type, variable, or function from a
	// file without build tags to one with build tags, or vice versa, will be
	// considered a breaking change; the recommended best practice is to only
	// place private types, variables, and functions in files with build tags.
	var prefix string
	if tags := buildTags(file); tags != "" {
		prefix = tags + "_"
	}
	decls := file.Decls
	for _, decl := range decls {
		if genDecl, isGenDecl := decl.(*ast.GenDecl); isGenDecl {
			switch genDecl.Tok {
			case token.TYPE:
				for _, spec := range genDecl.Specs {
					if typeSpec, isTypeSpec := spec.(*ast.TypeSpec); isTypeSpec {
						if name := typeSpec.Name.String(); ast.IsExported(name) {
							typeMap[prefix+name] = typeSpec
						}
					}
				}
			case token.CONST, token.VAR:
				for _, spec := range genDecl.Specs {
					if varSpec, isVarSpec := spec.(*ast.ValueSpec); isVarSpec {
						for _, ident := range varSpec.Names {
							if name := ident.String(); ast.IsExported(name) {
								varMap[prefix+name] = varSpec
							}
						}
					}
				}
			default:
				continue
			}
		} else if funcDecl, isFuncDecl := decl.(*ast.FuncDecl); isFuncDecl {
			if name := funcDecl.Name.String(); ast.IsExported(name) {
				// In defining a function, it's necessary to combine the Receiver and Name into a
				// Key because different types might have the same method names.
				var b strings.Builder
				// When the receiver of a method is a pointer, we will synchronously add this method
				// to the value type receiver of Receiver; this also means that if the receiver of a
				// method changes from a pointer receiver to a value receiver, it is considered as a
				// breaking change, but not vice versa.
				var ptrRecv bool
				if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
					for _, recv := range funcDecl.Recv.List {
						recvTypeExpr := recv.Type
					typeAssert:
						for {
							switch typeExpr := recvTypeExpr.(type) {
							case *ast.Ident:
								b.WriteString(typeExpr.String())
								b.WriteByte('_')
								break typeAssert
							case *ast.StarExpr:
								ptrRecv = true
								recvTypeExpr = typeExpr.X
							case *ast.SelectorExpr:
								recvTypeExpr = typeExpr.Sel
							case *ast.IndexExpr:
								recvTypeExpr = typeExpr.X
							case *ast.IndexListExpr:
								recvTypeExpr = typeExpr.X
							case *ast.ParenExpr:
								recvTypeExpr = typeExpr.X
							default:
								b.WriteString(formatExpr(typeExpr))
								b.WriteByte('_')
								break typeAssert
							}
						}
					}
				}
				b.WriteString(name)
				funcMap[prefix+b.String()] = funcDecl
				if ptrRecv {
					funcMap[prefix+"ptr_receiver_"+b.String()] = funcDecl
				}
			}
		}
	}
}

func buildTags(file *ast.File) string {
	const (
		goBuildPrefix   = "//go:build"
		plusBuildPrefix = "// +build"
	)
	tags := make([]string, 0, 2)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if tag, ok := strings.CutPrefix(comment.Text, goBuildPrefix); ok {
				tags = append(tags, strings.TrimSpace(tag))
			}
			if tag, ok := strings.CutPrefix(comment.Text, plusBuildPrefix); ok {
				tags = append(tags, strings.TrimSpace(tag))
			}
		}
	}
	if len(tags) == 0 {
		return ""
	}
	slices.Sort(tags)
	return strings.Map(
		func(r rune) rune {
			if unicode.IsSpace(r) {
				return '_'
			} else {
				return r
			}
		},
		strings.Join(tags, "_"),
	)
}

func typeDiff(oldType, newType *ast.TypeSpec) change {
	// Any changes to the generic parameters in the type definition will be considered as breaking changes.
	if genericChange := posFieldsDiff(oldType.TypeParams, newType.TypeParams); genericChange != noChange {
		return breakingChange
	}
	if (oldType.Assign != token.NoPos) != (newType.Assign != token.NoPos) {
		return breakingChange
	}
	if typeExprChange := typeExprDiff(oldType.Type, newType.Type); typeExprChange != noChange {
		return typeExprChange
	}
	return noChange
}

func posFieldsDiff(oldFields, newFields *ast.FieldList) change {
	if oldFields == nil && newFields == nil {
		return noChange
	}
	if oldFields != nil && newFields == nil {
		return breakingChange
	}
	if oldFields == nil {
		return breakingChange
	}
	if oldFields.NumFields() != newFields.NumFields() {
		return breakingChange
	}
	for i, oldField := range oldFields.List {
		newField := newFields.List[i]
		if typeExprChange := typeExprDiff(oldField.Type, newField.Type); typeExprChange != noChange {
			return breakingChange
		}
	}
	return noChange
}

func fieldsDiff(oldFields, newFields *ast.FieldList) change {
	if oldFields == nil && newFields == nil {
		return noChange
	}
	if oldFields != nil && newFields == nil {
		return breakingChange
	}
	if oldFields == nil {
		return somethingNew
	}
	var (
		oldFieldsMap = make(map[string]*ast.Field)
		newFieldsMap = make(map[string]*ast.Field)
	)
	for _, field := range oldFields.List {
		for _, name := range field.Names {
			oldFieldsMap[name.String()] = field
		}
	}
	for _, field := range newFields.List {
		for _, name := range field.Names {
			newFieldsMap[name.String()] = field
		}
	}
	for name, oldField := range oldFieldsMap {
		newField, ok := newFieldsMap[name]
		if !ok {
			return breakingChange
		}
		if fieldChange := fieldDiff(oldField, newField); fieldChange != noChange {
			return fieldChange
		}
	}
	if oldFields.NumFields() < newFields.NumFields() {
		return somethingNew
	}
	return noChange
}

func fieldDiff(oldField, newField *ast.Field) change {
	if typeExprChange := typeExprDiff(oldField.Type, newField.Type); typeExprChange != noChange {
		return typeExprChange
	}
	if tagChange := tagDiff(oldField.Tag, newField.Tag); tagChange != noChange {
		return tagChange
	}
	return noChange
}

func typeExprDiff(oldTypeExpr, newTypeExpr ast.Expr) change {
	if reflect.TypeOf(oldTypeExpr) != reflect.TypeOf(newTypeExpr) {
		return breakingChange
	}
	switch oldTypeExpr.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		if !isSameExpr(oldTypeExpr, newTypeExpr) {
			return breakingChange
		}
	case *ast.ParenExpr:
		oldParenExpr, newParenExpr := oldTypeExpr.(*ast.ParenExpr), newTypeExpr.(*ast.ParenExpr)
		if typeExprChange := typeExprDiff(oldParenExpr.X, newParenExpr.X); typeExprChange != noChange {
			return typeExprChange
		}
	case *ast.StarExpr:
		// For *ast.StarExpr, any changes will be considered as breaking changes.
		oldStarExpr, newStarExpr := oldTypeExpr.(*ast.StarExpr), newTypeExpr.(*ast.StarExpr)
		if typeExprChange := typeExprDiff(oldStarExpr.X, newStarExpr.X); typeExprChange != noChange {
			return breakingChange
		}
	case *ast.ArrayType:
		// For *ast.ArrayType, any changes will be considered as breaking changes.
		oldArrayType, newArrayType := oldTypeExpr.(*ast.ArrayType), newTypeExpr.(*ast.ArrayType)
		if !isSameExpr(oldArrayType.Len, newArrayType.Len) {
			return breakingChange
		}
		if typeExprChange := typeExprDiff(oldArrayType.Elt, newArrayType.Elt); typeExprChange != noChange {
			return breakingChange
		}
	case *ast.StructType:
		// For *ast.StructType:
		//  - deleting public fields is considered a breaking change;
		//  - changing the type of public fields is considered a breaking change;
		//  - deleting or changing private fields is not considered a breaking change;
		//  - adding any field is not considered a breaking change;
		//  - changing tag information is not considered a breaking change;
		oldStructType, newStructType := oldTypeExpr.(*ast.StructType), newTypeExpr.(*ast.StructType)
		if oldStructType.Incomplete != newStructType.Incomplete {
			return breakingChange
		}
		if fieldsChange := fieldsDiff(oldStructType.Fields, newStructType.Fields); fieldsChange != noChange {
			return fieldsChange
		}
	case *ast.FuncType:
		// For *ast.FuncType, any changes will be considered as breaking changes.
		oldFuncType, newFuncType := oldTypeExpr.(*ast.FuncType), newTypeExpr.(*ast.FuncType)
		if fieldsChange := posFieldsDiff(oldFuncType.TypeParams, newFuncType.TypeParams); fieldsChange != noChange {
			return breakingChange
		}
		if fieldsChange := posFieldsDiff(oldFuncType.Params, newFuncType.Params); fieldsChange != noChange {
			return breakingChange
		}
		if fieldsChange := posFieldsDiff(oldFuncType.Results, newFuncType.Results); fieldsChange != noChange {
			return breakingChange
		}
	case *ast.InterfaceType:
		// For *ast.InterfaceType, any changes at the method definition level will be considered a breaking change.
		oldInterfaceType, newInterfaceType := oldTypeExpr.(*ast.InterfaceType), newTypeExpr.(*ast.InterfaceType)
		if oldInterfaceType.Incomplete != newInterfaceType.Incomplete {
			return breakingChange
		}
		if fieldsChange := fieldsDiff(oldInterfaceType.Methods, newInterfaceType.Methods); fieldsChange != noChange {
			return breakingChange
		}
	case *ast.MapType:
		// For *ast.MapType, any changes in the Key and Value types will be considered a breaking change.
		oldMapType, newMapType := oldTypeExpr.(*ast.MapType), newTypeExpr.(*ast.MapType)
		if typeExprChange := typeExprDiff(oldMapType.Key, newMapType.Key); typeExprChange != noChange {
			return breakingChange
		}
		if typeExprChange := typeExprDiff(oldMapType.Value, newMapType.Value); typeExprChange != noChange {
			return breakingChange
		}
	case *ast.ChanType:
		// For *ast.ChanType, any changes will be considered as breaking changes.
		oldChanType, newChanType := oldTypeExpr.(*ast.ChanType), newTypeExpr.(*ast.ChanType)
		if (oldChanType.Arrow != token.NoPos) != (newChanType.Arrow != token.NoPos) {
			return breakingChange
		}
		if oldChanType.Dir != newChanType.Dir {
			return breakingChange
		}
		if typeExprChange := typeExprDiff(oldChanType.Value, newChanType.Value); typeExprChange != noChange {
			return breakingChange
		}
	default:
		if !isSameExpr(oldTypeExpr, newTypeExpr) {
			return breakingChange
		}
	}
	return noChange
}

func isSameExpr(oldExpr, newExpr ast.Expr) bool {
	var (
		oldExprBuf bytes.Buffer
		newExprBuf bytes.Buffer
	)
	fset := token.NewFileSet()
	printer.Fprint(&oldExprBuf, fset, oldExpr)
	printer.Fprint(&newExprBuf, fset, newExpr)
	return bytes.Equal(oldExprBuf.Bytes(), newExprBuf.Bytes())
}

func tagDiff(oldTag, newTag *ast.BasicLit) change {
	if oldTag == nil && newTag == nil {
		return noChange
	}
	if oldTag != nil && newTag == nil {
		return breakingChange
	}
	if oldTag == nil {
		return somethingNew
	}
	if oldTag.Value != newTag.Value {
		return somethingNew
	}
	return noChange
}

func formatExpr(expr ast.Expr) string {
	var formatter strings.Builder
	printer.Fprint(&formatter, token.NewFileSet(), expr)
	return formatter.String()
}
