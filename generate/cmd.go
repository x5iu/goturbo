package generate

import generateCmd "github.com/x5iu/genx/cmd"

var Command = generateCmd.Generate

func init() {
	Command.Use = "generate"
}
