package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var AppVersion string
var Environment string
var PrimaryRegions []string
var RegionalRegions []string
var DryRun bool
var SelfDestroy bool
var Account string
var LogLevel string
var Interactive bool
var Container string
var Namespace string
var DeploymentRing string
var Local bool
var Runner string
var PullRequest string
var StepWhitelist []string

// the base container for runiac
var DefaultBaseContainer = "runiac/deploy:latest-alpine-full"

func init() {
	deployCmd.Flags().StringVarP(&AppVersion, "version", "v", "", "Version of the iac code")
	deployCmd.Flags().StringVarP(&Environment, "environment", "e", "", "Targeted environment")
	deployCmd.Flags().StringVarP(&Account, "account", "a", "", "Targeted Cloud Account (ie. azure subscription, gcp project or aws account)")
	deployCmd.Flags().StringArrayVarP(&PrimaryRegions, "primary-regions", "p", []string{}, "Primary regions")
	deployCmd.Flags().StringArrayVarP(&RegionalRegions, "regional-regions", "r", []string{}, "Runiac will concurrently execute the ./regional directory across these regions setting the runiac_region input variable")
	deployCmd.Flags().BoolVar(&DryRun, "dry-run", false, "Dry Run")
	deployCmd.Flags().BoolVar(&SelfDestroy, "self-destroy", false, "Teardown after running deploy")
	deployCmd.Flags().StringVar(&LogLevel, "log-level", "", "Log level")
	deployCmd.Flags().BoolVar(&Interactive, "interactive", false, "Run Docker container in interactive mode")
	deployCmd.Flags().StringVarP(&Container, "container", "c", "", fmt.Sprintf("The runiac deploy container to execute in, defaults to '%s'", DefaultBaseContainer))
	deployCmd.Flags().StringVarP(&DeploymentRing, "deployment-ring", "d", "", "The deployment ring to configure")
	deployCmd.Flags().BoolVar(&Local, "local", false, "Pre-configure settings to create an isolated configuration specific to the executing machine")
	deployCmd.Flags().StringVarP(&Runner, "runner", "", "terraform", "The deployment tool to use for deploying infrastructure")
	deployCmd.Flags().StringSliceVarP(&StepWhitelist, "steps", "s", []string{}, "Only run the specified steps. To specify steps inside a track: -s {trackName}/{stepName}.  To run multiple steps, separate with a comma.  If empty, it will run all steps. To run no steps, specify a non-existent step.")
	deployCmd.Flags().StringVar(&PullRequest, "pull-request", "", "Pre-configure settings to create an isolated configuration specific to a pull request, provide pull request identifier")

	rootCmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy configurations",
	Long:  `This will execute the deploy action for each step.`,
	Run: func(cmd *cobra.Command, args []string) {
		checkDockerExists()

		ok := checkInitialized()
		if !ok {
			fmt.Printf("You need to run 'runiac init' before you can use the CLI in this directory\n")
			return
		}

		buildKit := "DOCKER_BUILDKIT=1"
		containerTag := viper.GetString("project")

		cmdd := exec.Command("docker", "build", "-t", containerTag, "-f", ".runiac/Dockerfile")

		cmdd.Args = append(cmdd.Args, getBuildArguments()...)

		var stdoutBuf, stderrBuf bytes.Buffer

		cmdd.Env = append(os.Environ(), buildKit)
		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Suffix = " Building project container..."
		s.Start()
		b, err := cmdd.CombinedOutput()
		if err != nil {
			s.Stop()
			logrus.Error(string(b))
			logrus.WithError(err).Fatalf("Building project container failed with %s\n", err)
		}
		s.Stop()

		logrus.Info("Completed build, lets run!")

		cmd2 := exec.Command("docker", "run", "--rm")

		cmd2.Env = append(os.Environ(), buildKit)

		// pre-configure for local development experience
		if Local {
			namespace, err := getMachineName()

			if err != nil {
				logrus.WithError(err).Fatal(err)
			}

			Namespace = namespace
			DeploymentRing = "local"
		} else if PullRequest != "" {
			Namespace = PullRequest
			DeploymentRing = "pr"
		}

		cmd2.Args = appendEIfSet(cmd2.Args, "DEPLOYMENT_RING", DeploymentRing)
		cmd2.Args = appendEIfSet(cmd2.Args, "RUNNER", Runner)
		cmd2.Args = appendEIfSet(cmd2.Args, "NAMESPACE", Namespace)
		cmd2.Args = appendEIfSet(cmd2.Args, "VERSION", AppVersion)
		cmd2.Args = appendEIfSet(cmd2.Args, "ENVIRONMENT", Environment)
		cmd2.Args = appendEIfSet(cmd2.Args, "DRY_RUN", fmt.Sprintf("%v", DryRun))
		cmd2.Args = appendEIfSet(cmd2.Args, "SELF_DESTROY", fmt.Sprintf("%v", SelfDestroy))
		cmd2.Args = appendEIfSet(cmd2.Args, "STEP_WHITELIST", strings.Join(StepWhitelist, ","))

		if len(PrimaryRegions) > 0 {
			cmd2.Args = appendEIfSet(cmd2.Args, "PRIMARY_REGION", PrimaryRegions[0])
		}

		if len(PrimaryRegions) > 0 {
			cmd2.Args = appendEIfSet(cmd2.Args, "REGIONAL_REGIONS", strings.Join(append(RegionalRegions, PrimaryRegions[0]), ","))
		}
		cmd2.Args = appendEIfSet(cmd2.Args, "ACCOUNT_ID", Account)
		cmd2.Args = appendEIfSet(cmd2.Args, "LOG_LEVEL", LogLevel)

		if Interactive {
			cmd2.Args = append(cmd2.Args, "-it")
		}

		// TODO: how to allow consumer whitelist environment variables or simply pass all in?
		for _, env := range cmd2.Env {
			if strings.HasPrefix(env, "TF_VAR_") {
				cmd2.Args = append(cmd2.Args, "-e", env)
			}
		}

		for _, env := range cmd2.Env {
			if strings.HasPrefix(env, "ARM_") {
				cmd2.Args = append(cmd2.Args, "-e", env)
			}
		}

		// handle local volume maps
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		// persist azure cli between container executions
		cmd2.Args = append(cmd2.Args, "-v", fmt.Sprintf("%s/.runiac/.azure:/root/.azure", dir))

		// persist gcloud cli
		cmd2.Args = append(cmd2.Args, "-v", fmt.Sprintf("%s/.runiac/.config/gcloud:/root/.config/gcloud", dir))

		// persist local terraform state between container executions
		cmd2.Args = append(cmd2.Args, "-v", fmt.Sprintf("%s/.runiac/tfstate:/runiac/tfstate", dir))

		cmd2.Args = append(cmd2.Args, containerTag)

		logrus.Info(strings.Join(cmd2.Args, " "))

		cmd2.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd2.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		cmd2.Stdin = os.Stdin

		err2 := cmd2.Run()
		if err2 != nil {
			log.Fatalf("Running iac failed with %s\n", err2)
		}
	},
}

func appendEIfSet(slice []string, arg string, val string) []string {
	if val != "" {
		return appendE(slice, arg, val)
	} else {
		return slice
	}
}
func appendE(slice []string, arg string, val string) []string {
	return append(slice, "-e", fmt.Sprintf("RUNIAC_%s=%s", arg, val))
}

func checkDockerExists() {
	_, err := exec.LookPath("docker")
	if err != nil {
		fmt.Printf("please add 'docker' to the path\n")
	}
}

func checkInitialized() bool {
	return InitAction()
}

func getBuildArguments() (args []string) {
	// check viper configuration if not set
	if Container == "" && viper.GetString("container") != "" {
		Container = viper.GetString("container")
	}

	if Container != "" {
		args = append(args, "--build-arg", fmt.Sprintf("RUNIAC_CONTAINER=%s", Container))
	}

	// must be last argument added for docker build current directory context
	args = append(args, ".")

	return
}

func getMachineName() (string, error) {
	// This handles most *nix platforms
	username := os.Getenv("USER")
	if username != "" {
		return sanitizeMachineName(username), nil
	}

	// This handles Windows platforms
	username = os.Getenv("USERNAME")
	if username != "" {
		return sanitizeMachineName(username), nil
	}

	// This is for other platforms without ENV vars set above
	cmdd := exec.Command("whoami")

	stdout, err := cmdd.StdoutPipe()
	if err != nil {
		return "", err
	}

	err = cmdd.Start()
	if err != nil {
		return "", err
	}

	out, err := ioutil.ReadAll(stdout)

	if err := cmdd.Wait(); err != nil {
		return "", err
	}

	return sanitizeMachineName(string(out)), err
}

func sanitizeMachineName(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile("[^a-zA-Z0-9]")

	return re.ReplaceAllLiteralString(s, "_")
}
