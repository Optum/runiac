package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "runiac",
	Short: "Runiac is a friendly runner for infrastructure as code",
	Long: `A friendly, portable code runner built with
love by Optum, tiny-dancer and friends.
Complete documentation is available at https://github.com/optum/runiac`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
