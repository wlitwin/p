package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionShort bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version, commit, and build info",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if versionShort {
			fmt.Println(Version)
			return
		}
		fmt.Printf("p %s\n", Version)
		fmt.Printf("  commit:  %s\n", GitCommit)
		fmt.Printf("  built:   %s\n", BuildDate)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionShort, "short", "s", false, "Print only the version number")
	rootCmd.AddCommand(versionCmd)
}
