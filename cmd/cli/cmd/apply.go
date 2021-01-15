package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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

func init() {
	applyCmd.Flags().StringVarP(&Version, "version", "v", "", "Version of the iac code")
	applyCmd.Flags().StringVarP(&Environment, "environment", "e", "", "Targeted environment")
	applyCmd.Flags().StringVarP(&Account, "account", "a", "", "Targeted Cloud Account (ie. azure subscription, gcp project)")
	applyCmd.Flags().StringArrayVarP(&PrimaryRegions, "primary-regions", "p", []string{}, "Primary regions")
	applyCmd.Flags().StringArrayVarP(&RegionalRegions, "regional-regions", "r", []string{}, "Regional regions")
	applyCmd.Flags().BoolVar(&DryRun, "dry-run", false, "Dry Run")
	applyCmd.Flags().BoolVar(&SelfDestroy, "self-destroy", false, "Teardown after running deploy")
	applyCmd.Flags().StringVar(&LogLevel, "log-level", "", "Log level")
	applyCmd.Flags().BoolVar(&Interactive, "interactive", false, "Run Docker container in interactive mode")

	rootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Execute apply across each step",
	Long:  `Execute apply across each step`,
	Run: func(cmd *cobra.Command, args []string) {
		checkDockerExists()

		ok := checkInitialized()
		if !ok {
			fmt.Printf("You need to run 'terrascale init' before you can use the CLI in this directory\n")
			return
		}

		buildKit := "DOCKER_BUILDKIT=1"
		containerTag := "sample"

		cmdd := exec.Command("docker", "build", "-t", containerTag, "-f", ".terrascalecli/Dockerfile", ".")

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

		if Version != "" {
			cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("VERSION=%s", Version))
		}

		if Environment != "" {
			cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("TERRASCALE_ENVIRONMENT=%s", Environment))
		}

		cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("TERRASCALE_DRY_RUN=%v", DryRun))
		cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("TERRASCALE_SELF_DESTROY=%v", SelfDestroy))

		if len(PrimaryRegions) > 0 {
			cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("TERRASCALE_PRIMARY_REGION=%s", PrimaryRegions[0]))
		}

		if len(RegionalRegions) > 0 || len(PrimaryRegions) > 0 {
			cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("TERRASCALE_REGIONAL_REGIONS=%s", strings.Join(append(RegionalRegions, PrimaryRegions[0]), ",")))
		}

		if Account != "" {
			cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("TERRASCALE_ACCOUNT_ID=%s", Account))
		}

		if LogLevel != "" {
			cmd2.Args = append(cmd2.Args, "-e", fmt.Sprintf("TERRASCALE_LOG_LEVEL=%s", LogLevel))
		}

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
		cmd2.Args = append(cmd2.Args, "-v", fmt.Sprintf("%s/.terrascalecli/.azure:/root/.azure", dir))
		
		// persist gcloud cli
		cmd2.Args = append(cmd2.Args, "-v", fmt.Sprintf("%s/.terrascalecli/.config/gcloud:/root/.config/gcloud", dir))

		// persist local terraform state between container executions
		cmd2.Args = append(cmd2.Args, "-v", fmt.Sprintf("%s/.terrascalecli/tfstate:/tfstate", dir))

		cmd2.Args = append(cmd2.Args, containerTag)

		log.Printf("%s\n", strings.Join(cmd2.Args, " "))

		cmd2.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd2.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		cmd2.Stdin = os.Stdin

		err2 := cmd2.Run()
		if err2 != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err2)
		}

		// -e ACCOUNT_ID="$ACCOUNT_ID" \
		// -e ENVIRONMENT="$ENVIRONMENT" \
		// -e NAMESPACE="$NAMESPACE" \
		// -e TERRASCALE_REGION_GROUP="us" \
		// -e TERRASCALE_STEP_WHITELIST="$STEP_WHITELIST" \
		// -e TERRASCALE_DRY_RUN="$DRY_RUN" \
		// -e TERRASCALE_SELF_DESTROY="$SELF_DESTROY" \
		// -e LOG_LEVEL="$LOG_LEVEL" \
		// -e DEPLOYMENT_RING="$DEPLOYMENT_RING" \
		// -e TERRASCALE_REGION_GROUPS="{\"azu\":{\"uk\":[\"uksouth\",\"ukwest\"],\"eu\":[\"northeurope\",\"westeurope\"],\"us\":[\"southcentralus\", \"northcentralus\"]}}" \
		// $INTERACTIVE_FLAG tssample
	},
}

func checkDockerExists() {
	_, err := exec.LookPath("docker")
	if err != nil {
		fmt.Printf("please add 'docker' to the path\n")
	}
}

func checkInitialized() bool {
	fs := afero.NewOsFs()
	
	ok, err := afero.DirExists(fs, ".terrascalecli")
	if err != nil {
		log.Fatalf("Unable to determine if CLI directory exists: %v", err)
		return false
	}

	return ok
}
