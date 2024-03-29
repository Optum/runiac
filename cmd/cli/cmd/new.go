package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gookit/color"
	"github.com/hashicorp/go-getter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var templateUrl string
var scm string
var runiacGitHubOrg = "github.com/runiac"

func init() {
	newCmd.Flags().StringVarP(&templateUrl, "url", "", "", "URL to download project template")
	newCmd.Flags().StringVarP(&scm, "scm", "", "git", "Default initializes git, set to None to prevent this")

	rootCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Create a new runiac project",
	Long:  `Creates scaffolding for a new runiac project`,
	Args: func(cmd *cobra.Command, args []string) error {

		if len(args) > 0 {
			name := args[0]

			_, err := checkIfProjectDirIsFree(name)

			if err != nil {
				return err
			}
		}

		if len(args) > 1 {
			return errors.New("too many arguments provided")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		projectName := ""
		if len(args) > 0 {
			projectName = args[0]
		}

		err := process(projectName)
		if err != nil {
			logrus.Error(fmt.Sprintf("Did not create a new project: %s", err))
			return
		}

		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		codeColor := color.LightBlue.Render
		bold := color.Bold.Render

		fmt.Printf(`Your new runiac project %s has successfully been created
at %s

Start by going to the directory with

%s

Run a local deployment to the cloud with

%s

See all commands with

%s

Enjoy!
		`, bold(projectName),
			bold(path.Join(dir, projectName)),
			codeColor(fmt.Sprintf("cd %s", projectName)),
			codeColor("runiac deploy -a <my-cloud-account> --local"),
			codeColor("runiac help"),
		)
	},
}

type DirectoryLayoutType int

const (
	Simple DirectoryLayoutType = iota
	Tracks
	UnknownDirectoryLayoutType
)

func stringToDirectoryLayoutType(s string) (DirectoryLayoutType, error) {
	if s == "simple" {
		return Simple, nil
	} else if s == "tracks" {
		return Tracks, nil
	}

	return UnknownDirectoryLayoutType, errors.New("invalid layout type")
}

type ToolType int

const (
	AzureCLIToolType ToolType = iota
	GCloudToolType
	UnknownToolType
)

type ScmType int

const (
	None ScmType = iota
	Git
	UnknownScmType
)

type RunnerType int

const (
	UnknownRunner RunnerType = iota
	ArmRunnerType
	TerraformRunnerType
)

const gitIgnore = `# Generated by runiac CLI.
.DS_Store
.runiac
`

const entrypointScript = `# Generated by runiac CLI.
runiac
`

const runiacConfig = `# Generated by runiac CLI.
project: ${PROJECT_NAME}
primary_region: ${PRIMARY_REGION}
regional_regions: ${PRIMARY_REGION}
runner: ${RUNNER}
`

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

func initializeGit(name string, fs afero.Fs) error {
	err := checkForGit()
	if err != nil {
		return err
	}

	// check if a git repository has already been initialized
	exists, err := afero.Exists(fs, fmt.Sprintf("%s/.git", name))
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// initialize the repository
	cmd := exec.Command("git", "init", "-b", "main")
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

func initializeFiles(projectName string, fs afero.Fs) (err error) {
	err = afero.WriteFile(fs, fmt.Sprintf("%s/entrypoint.sh", projectName), []byte(entrypointScript), 0744)
	if err != nil {
		return err
	}

	return
}

func checkIfProjectDirIsFree(dir string) (bool, error) {
	// check if directory already exists
	exists, _ := afero.DirExists(afero.NewOsFs(), dir)
	if exists {
		return false, fmt.Errorf("a directory '%s' already exists or is not valid. Choose a different project name", dir)
	}

	return true, nil
}

func process(name string) error {
	// ask for project name and validate that a directory with that name doesn't already exist

	if name == "" {
		err := survey.AskOne(&survey.Input{
			Message: "Choose a name for your project:",
			Default: "my-runiac-project",
		}, &name, survey.WithValidator(func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return errors.New("invalid directory. Choose a different project name")
			}

			// check if directory already exists
			_, err := checkIfProjectDirIsFree(str)
			if err != nil {
				return err
			}

			return nil
		}))

		if err != nil {
			return err
		}
	}

	// if a template url was provided, we can skip prompting for a source
	if len(templateUrl) > 0 {
		return processUrl(name, templateUrl)
	}

	// ask for project template
	template := ""
	err := survey.AskOne(&survey.Select{
		Message: "What type of project do you want to create?",
		Options: []string{
			"azure-arm - Use ARM templates for Microsoft Azure",
			"azure-terraform - Use Terraform for Microsoft Azure",
			"gcp-terraform - Use Terraform for Google Cloud Platform",
			"kitchen-sink - Use Terraform across various cloud providers and services",
			"starter-url - Download a starter template from github",
			// "custom - Configure a custom project not based on a specific template",
		},
		Help: `A template provides standard directory structures as recommended by runiac developers. You can always migrate from one
type of project to another by rearranging your directories manually after the fact.`,
	}, &template, survey.WithValidator(survey.Required))

	if err != nil {
		return err
	}

	projectTemplate := strings.TrimSpace(strings.Split(template, " - ")[0])
	if projectTemplate == "starter-url" {
		return processThirdParty(name)
	} else if projectTemplate == "custom" {
		return processCustom(name)
	} else {
		return processPredefined(name, projectTemplate)
	}

}

func promptForConfirmation() (bool, error) {
	prompt := &survey.Confirm{
		Message: "Does everything look good?",
	}

	confirm := false
	err := survey.AskOne(prompt, &confirm)
	if err != nil {
		return false, nil
	}

	return confirm, nil
}

func processThirdParty(name string) error {
	source := ""
	err := survey.AskOne(&survey.Input{
		Message: "What is the URL for the template?",
		Help: `This is the URL where the project template is hosted. It must be accessible from your current network, and must not 
require authentication. 

When using GitHub:
You can provide a shorthand form, such as github.com/user/runiac-template instead. For subdirectories, use a double slash
to indicate the path under the GitHub repository: github.com/user/runiac-template//simple-variant.`,
	}, &source, survey.WithValidator(survey.Required))

	if err != nil {
		return err
	}

	return processUrl(name, source)
}

func processUrl(name string, templateUrl string) error {
	err := getter.Get(name, templateUrl)
	if err != nil {
		return err
	}

	if scm == "git" {
		err = initializeGit(name, afero.NewOsFs())
		if err != nil {
			return err
		}
	}

	return nil
}

func processPredefined(name string, template string) error {
	source := ""
	switch template {
	case "azure-arm":
		source = fmt.Sprintf("%s/runiac-starter-arm-azure-hello-world", runiacGitHubOrg)
	case "azure-terraform":
		source = fmt.Sprintf("%s/runiac-starter-terraform-azure-hello-world", runiacGitHubOrg)
	case "aws-terraform":
		source = fmt.Sprintf("%s/runiac-starter-terraform-aws-hello-world", runiacGitHubOrg)
	case "gcp-terraform":
		source = fmt.Sprintf("%s/runiac-starter-terraform-gcp-hello-world", runiacGitHubOrg)
	case "kitchen-sink":
		source = "github.com/optum/runiac.git//examples/kitchen-sink"
	}

	err := getter.Get(name, source)
	if err != nil {
		return err
	}

	if scm == "git" {
		err = initializeGit(name, afero.NewOsFs())
		if err != nil {
			return err
		}
	}

	return nil
}

func processCustom(name string) error {
	questions := []*survey.Question{
		{
			Name: "layout",
			Prompt: &survey.Select{
				Message: "How do you want to organize your infrastructure deployment? (simple, tracks)",
				Options: []string{
					"simple - A single set of steps",
					"tracks - Multiple groups of steps that can be executed in parallel",
				},
				Help: `You can choose to organize your infrastructure deployment strategy into a single set of steps or into a more complex 
grouping of steps that can be executed in parallel.`,
			},
		},
		{
			Name: "primaryRegion",
			Prompt: &survey.Input{
				Message: "Which cloud provider region will your resources primarily be deployed to? (us-central1, southcentralus, etc.)",
				Help: `Depending on which cloud service(s) you intend to deploy to, this value will be the name of a region. For example, when deploying
to Microsoft Azure, valid regions include 'southcentralus' and 'eastus2'. When deploying to Google Cloud Platform, then the region
name could be 'us-central1'.`,
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
				Help: `runiac will invoke an underlying delivery tool to actually deploy your infrastructure. Currently, ARM templates and Terraform
are supported.`,
			},
		},
		{
			Name: "tools",
			Prompt: &survey.MultiSelect{
				Message: "Choose the set of tools you need to deploy your infrastructure:",
				Options: []string{
					"azure-cli - Microsoft Azure CLI",
					"gcloud - Google Cloud SDK",
				},
				Help: `Your infrastructure may require extra tooling apart from the underlying delivery tool. runiac providers some standard cloud
tools to facilitate this. You may choose zero or many tools to include in your project.`,
			},
		},
	}

	answers := struct {
		Layout        string
		PrimaryRegion string
		Runner        string
		Tools         []string
	}{}

	err := survey.Ask(questions, &answers)
	if err != nil {
		return err
	}

	layout, err := stringToDirectoryLayoutType(strings.TrimSpace(strings.Split(answers.Layout, " - ")[0]))
	if err != nil {
		return err
	}

	region := answers.PrimaryRegion
	runner := strings.TrimSpace(strings.Split(answers.Runner, " - ")[0])

	confirm, err := promptForConfirmation()
	if !confirm || err != nil {
		err = errors.New("")
		return err
	}

	fs := afero.NewOsFs()

	// create the project directory
	err = fs.Mkdir(name, 0755)
	if err != nil {
		logrus.WithError(err).Error(err)
		return errors.New("")
	}

	// initialize default files
	err = initializeFiles(name, fs)
	if err != nil {
		logrus.WithError(err).Error(err)
		return errors.New("")
	}

	// create directory structures
	switch layout {
	case Simple:
		err = createSimpleDirectories(name, fs)
		if err != nil {
			logrus.WithError(err).Error(err)
			return errors.New("")
		}

	case Tracks:
		err = createTracksDirectories(name, fs)
		if err != nil {
			logrus.WithError(err).Error(err)
			return errors.New("")
		}
	}

	// initialize the runiac config file
	err = initializeRuniacConfig(name, region, runner, fs)
	if err != nil {
		logrus.WithError(err).Error(err)
		return err
	}

	// initialize scm repositories
	if scm == "git" {
		err = initializeGit(name, afero.NewOsFs())
		if err != nil {
			return err
		}
	}

	return nil
}
