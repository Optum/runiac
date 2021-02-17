package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type promptItem struct {
	Key         string
	Label       string
	Description string
}

var selectTemplates = &promptui.SelectTemplates{
	Label:    "{{ . }}?",
	Active:   "{{ .Label | green }} - {{ .Description }}",
	Inactive: "{{ .Label | white }} - {{ .Description }}",
	Selected: "{{ .Label | green }} - {{ .Description }}",
}

var projectTemplate string
var primaryRegion string
var scmType string
var runner string
var nonInteractive bool

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

type Runner int

const (
	UnknownRunner Runner = iota
	ArmRunner
	TerraformRunner
)

func stringToRunner(s string) (Runner, error) {
	if s == "arm" {
		return ArmRunner, nil
	} else if s == "terraform" {
		return TerraformRunner, nil
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

func promptForValues() (name string, err error) {
	prompt := promptui.Prompt{
		Label: "Project name",
		Validate: func(input string) error {
			if len(input) == 0 {
				return errors.New("Name must not be empty")
			} else if strings.Contains(input, " ") {
				return errors.New("Name cannot contain spaces")
			}

			return nil
		},
	}

	name, err = prompt.Run()
	if err != nil {
		return
	}

	projectTemplates := []promptItem{
		{Key: "simple", Label: "Simple", Description: "single set of steps"},
		{Key: "tracks", Label: "Tracks", Description: "multiple steps organized by parallel tracks"},
	}

	pSelect := promptui.Select{
		Label:     "What type of project do you want to create",
		Items:     projectTemplates,
		Templates: selectTemplates,
		Size:      len(projectTemplates),
	}

	i, _, err := pSelect.Run()
	if err != nil {
		return
	}

	projectTemplate = projectTemplates[i].Key

	prompt = promptui.Prompt{
		Label: "What cloud provider region will your infrastructure primarily be deployed to? (us-central1, southcentralus, etc.)",
	}

	primaryRegion, err = prompt.Run()
	if err != nil {
		return
	}

	scmTypes := discoverScms()
	scmOptions := make([]promptItem, 0)
	for _, item := range scmTypes {
		switch item {
		case Git:
			scmOptions = append(scmOptions, promptItem{
				Key:         "git",
				Label:       "git",
				Description: "initialize a git repository in the project directory",
			})
		}
	}

	scmOptions = append(scmOptions, promptItem{
		Key:         "none",
		Label:       "None",
		Description: "do not initialize any source control repository",
	})

	pSelect = promptui.Select{
		Label:     "What source control tool do you want to use",
		Items:     scmOptions,
		Templates: selectTemplates,
		Size:      len(scmOptions),
	}

	i, _, err = pSelect.Run()
	if err != nil {
		return
	}

	scmType = scmOptions[i].Key

	runners := []promptItem{
		{Key: "arm", Label: "ARM Templates", Description: "use Azure Resource Manager deployment templates (preview)"},
		{Key: "terraform", Label: "Terraform", Description: "use Terraform"},
	}

	pSelect = promptui.Select{
		Label:     "What delivery tool do you want to use",
		Items:     runners,
		Templates: selectTemplates,
		Size:      len(runners),
	}

	i, _, err = pSelect.Run()
	if err != nil {
		return
	}

	runner = runners[i].Key

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

			name, err = promptForValues()
			if err != nil {
				logrus.Error(fmt.Sprintf("CLI terminated: %v", err))
				return
			}
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

		fmt.Printf("🍺 Initialized a new project. You can now run the 'runiac init' command in the %s directory.\n", name)
	},
}
