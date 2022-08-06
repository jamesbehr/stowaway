package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "stowaway",
	Short: "Symlink farm manager",
}

func init() {
	rootCmd.AddCommand(stowCmd)
	rootCmd.AddCommand(packagesCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
