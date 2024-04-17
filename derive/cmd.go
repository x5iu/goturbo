package derive

import (
	"github.com/spf13/cobra"
	"github.com/x5iu/goturbo/derive/lombok"
)

var Command = &cobra.Command{
	Use: "derive",
}

func init() {
	Command.AddCommand(lombok.Command)
}
