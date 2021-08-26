package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)


func init() {
	rootCmd.AddCommand(ejectCmd)
}

var ejectCmd = &cobra.Command{
	Use:   "eject [feature]",
	Short: "Eject a specific feature",
	Long:  `Take control over specific features typically handled by runiac.  Currently supports runiac eject dockerfile`,
	Args: func(cmd *cobra.Command, args []string) error {

		if len(args) > 0 {
			name := args[0]

			if !checkIfEjectArgIsSupported(name) {
				return fmt.Errorf("%s is not a supported argument, only dockerfile is supported", name)
			}
		}

		if len(args) > 1 {
			return errors.New("too many arguments provided")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := writeDockerfile(afero.NewOsFs())
		if err != nil {
			return
		}

		viper.Set("dockerfile", "Dockerfile")
		_ = viper.WriteConfig()

	},
}

func checkIfEjectArgIsSupported(arg string) bool {
	return strings.ToLower(arg) == "dockerfile"
}
func writeDockerfile(fs afero.Fs) (err error) {
	err = afero.WriteFile(fs,"Dockerfile", []byte(DockerfileTemplate), 0744)
	if err != nil {
		return err
	}

	return
}
