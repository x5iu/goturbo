package upgrade

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
)

var (
	file string
)

var Command = &cobra.Command{
	Use:     "upgrade version",
	Version: "v0.1.1",
	Short:   "A tool used to determine the next semantic version.",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 && file == "" {
			return cmd.Usage()
		}
		var old SemanticVersion
		if len(args) > 0 {
			version := args[0]
			old, err = parse(version)
			if err != nil {
				return err
			}
		}
		chg, err := detectChange()
		if err != nil {
			return err
		}
		if file != "" {
			if err = inferUpdate(file, old, chg); err != nil {
				return err
			}
		} else {
			next := old.Next(chg)
			fmt.Println(next)
		}
		return nil
	},
}

func init() {
	flags := Command.PersistentFlags()
	flags.StringVarP(&file, "file", "f", "", "version file, the version number will be automatically inferred from the file and updated")
}

func inferUpdate(file string, old SemanticVersion, chg change) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	ast.Inspect(f, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				lit, _ := strconv.Unquote(x.Value)
				if lit != "" {
					if sv, _ := parse(lit); sv.Valid() {
						if old.Valid() {
							sv = old
						}
						x.Value = strconv.Quote(sv.Next(chg).String())
					}
				}
				return false
			}
		default:
		}
		return true
	})
	var buf bytes.Buffer
	if err = format.Node(&buf, fset, f); err != nil {
		return err
	}
	if err = os.WriteFile(file, buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}
