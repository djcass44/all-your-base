package cmd

import "github.com/spf13/cobra"

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build an image",
}
