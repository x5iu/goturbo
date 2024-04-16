package main

import (
	"github.com/spf13/cobra"
	"github.com/x5iu/goturbo/generate"
)

var GoTurbo = &cobra.Command{
	Use:           "goturbo",
	Version:       "v0.0.1",
	Short:         "a toolkit designed to enhance the efficiency of Golang development",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	GoTurbo.AddCommand(generate.Command)
}

func main() {
	cobra.CheckErr(GoTurbo.Execute())
}
