package lombok

import (
	"strconv"

	lombokPkg "github.com/x5iu/visc/cmd"
)

var Command = lombokPkg.Command

func init() {
	// Redefine these global variables to make them more closely aligned with the `goturbo` program.
	lombokPkg.ProgramName = "derive"
	lombokPkg.GeneratorName = strconv.Quote("goturbo derive")
	lombokPkg.DirectivePrefix = "@derive."

	// Drawing inspiration from the name of Project Lombok in Java, `visc` performs tasks somewhat similar to
	// those of Project Lombok.
	Command.Use = "lombok"
	Command.Short = "Somewhat similar to Java's Project Lombok, it generates getters/setters/constructors for structs."
}
