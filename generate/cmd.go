package generate

import generateCmd "github.com/x5iu/genx/cmd"

var Command = generateCmd.Generate

func init() {
	// v0.0.2: The original command name was `genx`, but in goturbo, to make the command more understandable,
	// the sub-command name was manually changed to `generate`.
	Command.Use = "generate"
	Command.Short = "Execute the `go generate` command for the entire project with a single command, no matter how deeply it is hidden."
}
