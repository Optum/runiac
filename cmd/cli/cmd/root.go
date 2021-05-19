package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "runiac",
	Short: "Runiac is a friendly runner for infrastructure as code",
	Long: `A friendly, portable infrastructure as code runner built with
love by tiny-dancer and friends. Open sourced for the community by Optum.
Complete documentation is available at https://runiac.io`,
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

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// viper.AddConfigPath(".")
	viper.SetConfigFile("runiac.yml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		//logrus.WithError(err).Warn("Failed reading .runiac configuration")
	}
}
