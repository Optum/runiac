package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var Version string
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
var PullRequest string

func init() {
	deployCmd.Flags().StringVarP(&Version, "version", "v", "", "Version of the iac code")
	deployCmd.Flags().StringVarP(&Environment, "environment", "e", "", "Targeted environment")
	deployCmd.Flags().StringVarP(&Account, "account", "a", "", "Targeted Cloud Account (ie. azure subscription, gcp project)")
	deployCmd.Flags().StringArrayVarP(&PrimaryRegions, "primary-regions", "p", []string{}, "Primary regions")
	deployCmd.Flags().StringArrayVarP(&RegionalRegions, "regional-regions", "r", []string{}, "Regional regions")
	deployCmd.Flags().BoolVar(&DryRun, "dry-run", false, "Dry Run")
	deployCmd.Flags().BoolVar(&SelfDestroy, "self-destroy", false, "Teardown after running deploy")
	deployCmd.Flags().StringVar(&LogLevel, "log-level", "", "Log level")
	deployCmd.Flags().BoolVar(&Interactive, "interactive", false, "Run Docker container in interactive mode")
	deployCmd.Flags().StringVarP(&Container, "container", "c", "runiac:alpine", "The container to execute, defaults 'runiac:alpine'")
	deployCmd.Flags().StringVarP(&DeploymentRing, "deployment-ring", "d", "", "The deployment ring to configure")
	deployCmd.Flags().BoolVar(&Local, "local", false, "Pre-configure settings to create an isolated configuration specific to the executing machine")
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
		containerTag := "sample"

		cmdd := exec.Command("docker", "build", "-t", containerTag, "-f", ".runiac/Dockerfile", "--build-arg", fmt.Sprintf("RUNIAC_CONTAINER=%s", Container), ".")

		var stdoutBuf, stderrBuf bytes.Buffer
		cmdd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmdd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

		cmdd.Env = append(os.Environ(), buildKit)

		err := cmdd.Run()
		if err != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}

		cmd2 := exec.Command("docker", "run")

		cmd2.Env = append(os.Environ(), buildKit)

		// pre-configure for local development experience
		if Local {
			namespace, err := getMachineName()

			if err != nil {
				log.Fatal(err)
			}

			Namespace = namespace
			DeploymentRing = "local"
		} else if PullRequest != "" {
			Namespace = PullRequest
			DeploymentRing = "pr"
		}

		cmd2.Args = appendEIfSet(cmd2.Args, "DEPLOYMENT_RING", DeploymentRing)
		cmd2.Args = appendEIfSet(cmd2.Args, "NAMESPACE", Namespace)
		cmd2.Args = appendEIfSet(cmd2.Args, "VERSION", Version)
		cmd2.Args = appendEIfSet(cmd2.Args, "ENVIRONMENT", Environment)
		cmd2.Args = appendEIfSet(cmd2.Args, "DRY_RUN", fmt.Sprintf("%v", DryRun))
		cmd2.Args = appendEIfSet(cmd2.Args, "SELF_DESTROY", fmt.Sprintf("%v", SelfDestroy))

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

		for _, env := range cmd2.Env {
			if strings.HasPrefix(env, "TF_VAR_") {
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
		cmd2.Args = append(cmd2.Args, "-v", fmt.Sprintf("%s/.runiac/tfstate:/tfstate", dir))

		cmd2.Args = append(cmd2.Args, containerTag)

		log.Printf("%s\n", strings.Join(cmd2.Args, " "))

		cmd2.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd2.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		cmd2.Stdin = os.Stdin

		err2 := cmd2.Run()
		if err2 != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err2)
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
	fs := afero.NewOsFs()

	ok, err := afero.DirExists(fs, ".runiac")
	if err != nil {
		log.Fatalf("Unable to determine if CLI directory exists: %v", err)
		return false
	}

	return ok
}

func getMachineName() (string, error) {
	if runtime.GOOS == "windows" {
		fmt.Println("Hello from Windows")
		return "", nil
	} else {
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

		return strings.TrimSuffix(fmt.Sprintf("%s", out), "\n"), err
	}

}
