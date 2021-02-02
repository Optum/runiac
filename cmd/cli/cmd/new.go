package cmd

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var projectTemplate string
var scmType string

type ProjectTemplateType int

const (
	Simple ProjectTemplateType = iota
	Tracks
	UnknownProjectTemplateType
)

func stringToTemplateType(s string) (ProjectTemplateType, error) {
	if s == "simple" {
		return Simple, nil
	} else if s == "tracks" {
		return Tracks, nil
	}

	return UnknownProjectTemplateType, errors.New("Invalid template type")
}

type ScmType int

const (
	None ScmType = iota
	Git
	UnknownScmType
)

func stringToScmType(s string) (ScmType, error) {
	if s == "none" {
		return None, nil
	} else if s == "git" {
		return Git, nil
	}

	return UnknownScmType, errors.New("Invalid SCM type")
}

const gitIgnore = `
.DS_Store
.runiac
`

func init() {
	newCmd.Flags().StringVar(&projectTemplate, "template", "simple", "Create scaffolding using a predefined project template (simple, tracks)")
	newCmd.Flags().StringVar(&scmType, "scm", "git", "Initialize a repository in the project directory (none, git)")

	rootCmd.AddCommand(newCmd)
}

func createSimpleDirectories(name string, fs afero.Fs) error {
	err := fs.MkdirAll(fmt.Sprintf("%s/step1_initial", name), 0755)
	if err != nil {
		return err
	}

	return nil
}

func createTracksDirectories(name string, fs afero.Fs) error {
	err := fs.MkdirAll(fmt.Sprintf("%s/tracks/initial/step1_initial", name), 0755)
	if err != nil {
		return err
	}

	return nil
}

func initializeGit(name string, fs afero.Fs) error {
	// check if git is available
	_, err := exec.LookPath("git")
	if err != nil {
		return nil
	}

	// initialize the repository
	cmd := exec.Command("git", "init")
	cmd.Dir = name
	_, err = cmd.Output()
	if err != nil {
		return err
	}

	// create a default .gitignore
	err = afero.WriteFile(fs, fmt.Sprintf("%s/.gitignore", name), []byte(gitIgnore), 0644)
	if err != nil {
		return err
	}

	return nil
}

var newCmd = &cobra.Command{
	Use:   "new <project-name>",
	Short: "Create a new Runiac project",
	Long:  `Creates scaffolding for a new Runiac project`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate template type
		template, err := stringToTemplateType(projectTemplate)
		if err != nil {
			logrus.Error(fmt.Sprintf("Unknown project template '%s' (valid types: simple, tracks)", projectTemplate))
			return
		}

		// validate scm type
		scm, err := stringToScmType(scmType)
		if err != nil {
			logrus.Error(fmt.Sprintf("Unknown SCM type '%s' (valid types: none, git)", scmType))
			return
		}

		name := args[0]
		fs := afero.NewOsFs()

		// check if directory already exists
		exists, _ := afero.DirExists(fs, name)
		if exists {
			logrus.Error(fmt.Sprintf("A directory '%s' already exists. Choose a different project name.", name))
			return
		}

		// create the project directory
		err = fs.Mkdir(name, 0755)
		if err != nil {
			logrus.WithError(err).Error(err)
			return
		}

		// create directory structures
		switch template {
		case Simple:
			err = createSimpleDirectories(name, fs)
			if err != nil {
				logrus.WithError(err).Error(err)
				return
			}

			break

		case Tracks:
			err = createTracksDirectories(name, fs)
			if err != nil {
				logrus.WithError(err).Error(err)
				return
			}

			break
		}

		// initialize scm repositories
		switch scm {
		case Git:
			err = initializeGit(name, fs)
			if err != nil {
				logrus.WithError(err).Error(err)
			}

			break
		}

		fmt.Printf("Initialized a new project. You can now run the 'runiac init' command in the %s directory.\n", name)
	},
}
