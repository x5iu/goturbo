package merge

import "github.com/spf13/cobra"

var (
	output string
)

var Command = &cobra.Command{
	Use:     "merge",
	Version: "v0.0.1",
	Short:   "Merge multiple `.go` files, suitable for streamlining the results of code generation.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return merge(cmd, args, output)
	},
}

func init() {
	Command.PersistentFlags().StringVar(&output, "output", "", "output file name")
	Command.MarkPersistentFlagRequired("output")
}
