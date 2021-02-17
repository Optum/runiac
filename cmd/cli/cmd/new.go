package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var projectTemplate string
var primaryRegion string
var scmType string
var runner string
var tools string
var nonInteractive bool

type surveyAnswers struct {
	Name            string
	ProjectTemplate string
	PrimaryRegion   string
	Runner          string
	Tools           []string
	Scm             string
}

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

type ToolType int

const (
	AzureToolType ToolType = iota
	GCloudToolType
	UnknownToolType
)

func stringToToolType(s string) (ToolType, error) {
	if s == "azure" {
		return AzureToolType, nil
	} else if s == "gcloud" {
		return GCloudToolType, nil
	}

	return UnknownToolType, errors.New("Invalid tool type")
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

type RunnerType int

const (
	UnknownRunner RunnerType = iota
	ArmRunnerType
	TerraformRunnerType
)

func stringToRunner(s string) (RunnerType, error) {
	if s == "arm" {
		return ArmRunnerType, nil
	} else if s == "terraform" {
		return TerraformRunnerType, nil
	}

	return UnknownRunner, errors.New("Invalid runner type")
}

const gitIgnore = `
.DS_Store
.runiac
`

const runiacConfig = `project: ${PROJECT_NAME}
primary_region: ${PRIMARY_REGION}
regional_regions: ${PRIMARY_REGION}
runner: ${RUNNER}
`

func init() {
	newCmd.Flags().StringVar(&projectTemplate, "template", "", "Create scaffolding using a predefined project template (simple, tracks)")
	newCmd.Flags().StringVar(&scmType, "scm", "", "Initialize a repository in the project directory (none, git)")
	newCmd.Flags().StringVar(&primaryRegion, "primary-region", "", "Primary cloud provider region where you intend to deploy infrastructure")
	newCmd.Flags().StringVar(&runner, "runner", "", "Configures the project for a specific deployment tool (arm, terraform)")
	newCmd.Flags().StringVar(&runner, "tools", "", "Comma-separated list of tools to include in the project (azure, gcloud)")
	newCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Disables manual input prompts")

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

func checkForGit() (err error) {
	// check if git is available
	_, err = exec.LookPath("git")
	return
}

func discoverScms() (scmTypes []ScmType) {
	scmTypes = make([]ScmType, 0)

	err := checkForGit()
	if err == nil {
		scmTypes = append(scmTypes, Git)
	}

	return
}

func initializeGit(name string, fs afero.Fs) error {
	err := checkForGit()
	if err != nil {
		return err
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

func initializeRuniacConfig(projectName string, region string, runner string, fs afero.Fs) (err error) {
	runiacConfigYml := strings.ReplaceAll(runiacConfig, "${PROJECT_NAME}", projectName)
	runiacConfigYml = strings.ReplaceAll(runiacConfigYml, "${PRIMARY_REGION}", region)
	runiacConfigYml = strings.ReplaceAll(runiacConfigYml, "${RUNNER}", runner)

	err = afero.WriteFile(fs, fmt.Sprintf("%s/runiac.yml", projectName), []byte(runiacConfigYml), 0644)
	if err != nil {
		return err
	}

	return
}

func promptForValues() (result surveyAnswers, err error) {
	scmTypes := discoverScms()
	scmOptions := make([]string, 0)
	for _, scmType := range scmTypes {
		switch scmType {
		case Git:
			scmOptions = append(scmOptions, "git - Initialize a git repository in the project directory")
		}
	}

	scmOptions = append(scmOptions, "None - do not use any source control tool")

	answers := surveyAnswers{}
	questions := []*survey.Question{
		{
			Name:     "name",
			Prompt:   &survey.Input{
				Message: "Choose a name for your project.",
			},
			Validate: survey.Required,
		},
		{
			Name:     "primaryRegion",
			Prompt:   &survey.Input{
				Message: "Which cloud provider region will your resources primarily be deployed to? (us-central1, southcentralus, etc.)",
			},
		},
		{
			Name: "projectTemplate",
			Prompt: &survey.Select{
				Message: "What type of project do you want to create?",
				Options: []string{
					"simple - A single set of deployment steps", 
					"tracks - Multiple sets of steps that can be deployed in parallel",
				},
			},
		},
		{
			Name: "runner",
			Prompt: &survey.Select{
				Message: "Which deployment tool do you want to use?",
				Options: []string{
					"arm - Use Azure Resource Manager templates (preview)", 
					"terraform - Use Hashicorp Terraform",
				},
			},
		},
		{
			Name: "tools",
			Prompt: &survey.MultiSelect{
				Message: "Choose the set of tools you need to deploy your infrastructure:",
				Options: []string{
					"azure - Microsoft Azure CLI", 
					"gcloud - Google Cloud SDK",
				},
			},
		},
		{
			Name: "scm",
			Prompt: &survey.Select{
				Message: "Which source control tool do you want to use?",
				Options: scmOptions,
			},
		},
	}

	err = survey.Ask(questions, &answers)
	if err != nil {
		return
	}

	confirm := false
	prompt := &survey.Confirm{
		Message: "Does everything look good?",
	}

	survey.AskOne(prompt, &confirm)
	if !confirm {
		err = errors.New("")
		return
	}

	result = surveyAnswers{
		Name: answers.Name,
		PrimaryRegion: answers.PrimaryRegion,
		ProjectTemplate: strings.TrimSpace(strings.Split(answers.ProjectTemplate, "-")[0]),
		Runner: strings.TrimSpace(strings.Split(answers.Runner, "-")[0]),
		Scm: strings.TrimSpace(strings.Split(answers.Scm, "-")[0]),
	}

	result.Tools = make([]string, 0)
	for _, tool := range answers.Tools {
		result.Tools = append(result.Tools, strings.TrimSpace(strings.Split(tool, "-")[0]))
	}

	return
}

var newCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Create a new runiac project",
	Long:  `Creates scaffolding for a new runiac project`,
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		var name string
		var err error
		if len(args) > 0 {
			name = args[0]
		}

		// no arguments given, initiate a prompt for manual input
		if len(args) == 0 && projectTemplate == "" && primaryRegion == "" && scmType == "" && runner == "" {
			// unless the cli is specifically set not to do so
			if nonInteractive {
				logrus.Error("Not enough arguments given, but CLI is in non-interactive mode")
				return
			}

			result, err := promptForValues()
			if err != nil {
				logrus.Error(fmt.Sprintf("CLI terminated: %v", err))
				return
			}

			name = result.Name
			projectTemplate = result.ProjectTemplate
			primaryRegion = result.PrimaryRegion
			runner = result.Runner
			scmType = result.Scm
			tools = strings.Join(result.Tools, ",")
		}

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

		// validate container tools
		containerTools := make([]ToolType, 0)
		for _, toolType := range strings.Split(tools, ",") {
			if toolType == "azure" {
				containerTools = append(containerTools, AzureToolType)
			} else if toolType == "gcloud" {
				containerTools = append(containerTools, GCloudToolType)
			} else {
				logrus.Error(fmt.Sprintf("Unknown tool type '%s' (valid types: azure, gcloud)", toolType))
			}
		}

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

		// initialize the runiac config file
		err = initializeRuniacConfig(name, primaryRegion, runner, fs)
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

		fmt.Printf("🍺 Initialized a new project in directory: %s.\n", name)
	},
}
