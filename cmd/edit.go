package cmd

import (
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Deterministic edit primitives for todos and knowledge",
}

func init() {
	rootCmd.AddCommand(editCmd)
}
