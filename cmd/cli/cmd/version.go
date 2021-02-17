package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of runiac",
	Long:  `All software has versions. This is runiac's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("runiac v0.0.1-beta4")
	},
}
