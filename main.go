package main

import (
	"github.com/spf13/cobra"
	"github.com/x5iu/goturbo/derive"
	"github.com/x5iu/goturbo/generate"
	"github.com/x5iu/goturbo/merge"
	"github.com/x5iu/goturbo/upgrade"
)

var GoTurbo = &cobra.Command{
	Use:           "goturbo",
	Version:       "v0.0.6",
	Short:         "a toolkit designed to enhance the efficiency of Golang development",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	GoTurbo.AddCommand(generate.Command)
	GoTurbo.AddCommand(derive.Command)
	GoTurbo.AddCommand(upgrade.Command)
	GoTurbo.AddCommand(merge.Command)
}

func main() {
	cobra.CheckErr(GoTurbo.Execute())
}
