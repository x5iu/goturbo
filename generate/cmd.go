package generate

import generateCmd "github.com/x5iu/genx/cmd"

var Command = generateCmd.Generate

func init() {
	// v0.0.2: The original command name was `genx`, but in goturbo, to make the command more understandable,
	// the sub-command name was manually changed to `generate`.
	Command.Use = "generate"
}
