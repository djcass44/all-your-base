package cache

import "github.com/spf13/cobra"

var Command = &cobra.Command{
	Use:   "cache",
	Short: "Cache utilities",
}

func init() {
	Command.AddCommand(cleanCmd)
}
