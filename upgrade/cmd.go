package upgrade

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:     "upgrade version",
	Version: "v0.0.1",
	Short:   "A tool used to determine the next semantic version.",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		semanticVersion, err := parse(version)
		if err != nil {
			return err
		}
		chg, err := detectChange()
		if err != nil {
			return err
		}
		nextVersion := semanticVersion.Next(chg)
		fmt.Println(nextVersion)
		return nil
	},
}
