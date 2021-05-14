package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version, Commit, Date string

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of runiac",
	Long:  `All software has versions. This is runiac's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("runiac %s. Commit %s.  Built on %s.\n", Version, Commit, Date)
	},
}
