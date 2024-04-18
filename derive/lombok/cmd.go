package lombok

import lombokPkg "github.com/x5iu/visc/cmd"

var Command = lombokPkg.Command

func init() {
	// Drawing inspiration from the name of Project Lombok in Java, `visc` performs tasks somewhat similar to
	// those of Project Lombok.
	Command.Use = "lombok"
	Command.Short = "Somewhat similar to Java's Project Lombok, it generates getters/setters/constructors for structs."
}
