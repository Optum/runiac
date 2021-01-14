package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"github.optum.com/healthcarecloud/terrascale/pkg/logging"
	"github.optum.com/healthcarecloud/terrascale/pkg/steps"
	"github.optum.com/healthcarecloud/terrascale/pkg/terraform"
	"github.optum.com/healthcarecloud/terrascale/pkg/tracks"
)

var fs afero.Fs
var tracker tracks.Tracker
var deployment config.Deployment
var log *logrus.Entry

func main() {
	initFunc()

	log.Debugf("Beginning Account Deployment: %s", deployment.Config.AccountID)

	log.Debug("Executing tracks...")

	// assume terraform for all steps
	output := tracker.ExecuteTracks(steps.TerraformOnlyStepperFactory{}, deployment.Config)

	log.Debug("Completed executing tracks...")

	trackCount := len(output.Tracks)
	failedSteps := []string{}
	skippedSteps := []string{}
	skippedTracks := []string{}
	failedDestroySteps := []string{}
	stepCount := 0
	executedStepCount := 0
	failedTestCount := 0

	for _, t := range output.Tracks {
		if t.Skipped {
			skippedTracks = append(skippedTracks, t.Name)
		}

		for _, tExecution := range t.Output.Executions {
			executedStepCount += tExecution.Output.ExecutedCount
			stepCount += tExecution.Output.ExecutedCount + tExecution.Output.SkippedCount
			failedTestCount += tExecution.Output.FailedTestCount

			for _, s := range tExecution.Output.Steps {
				switch s.Output.Status {
				case steps.Fail:
					failedSteps = append(failedSteps, fmt.Sprintf("%v/%v/%v/%v", t.Name, s.Name, tExecution.RegionDeployType, tExecution.Region))
				case steps.Skipped:
					skippedSteps = append(skippedSteps, fmt.Sprintf("%v/%v/%v/%v", t.Name, s.Name, tExecution.RegionDeployType, tExecution.Region))
				}
			}

		}

		for _, tExecution := range t.DestroyOutput.Executions {
			for _, fStep := range tExecution.Output.FailedSteps {
				failedDestroySteps = append(failedDestroySteps, fmt.Sprintf("%v/%v/%v/%v", t.Name, fStep.Name, tExecution.RegionDeployType, tExecution.Region))
			}
		}
	}

	failedStepCount := len(failedSteps)

	resultMessage := fmt.Sprintf("Executed %v/%v steps successfully with %v test failure(s) across %v track(s).",
		executedStepCount-failedStepCount, stepCount, failedTestCount, trackCount-len(skippedTracks))

	result := "success"

	if failedStepCount > 0 {
		resultMessage += fmt.Sprintf("  Failed: %v.", strings.Join(failedSteps, ", "))
		result = "fail"
	}

	if len(skippedSteps) > 0 {
		resultMessage += fmt.Sprintf("  Skipped: %v.", strings.Join(skippedSteps, ", "))
		result = "fail"
	}

	if len(failedDestroySteps) > 0 {
		resultMessage += fmt.Sprintf("  Failed to destroy: %v.", strings.Join(failedDestroySteps, ", "))
		result = "fail"
	}

	slog := log.WithFields(logrus.Fields{
		"type":          "summary",
		"skipped":       strings.Join(skippedSteps, ","),
		"failed":        strings.Join(failedSteps, ","),
		"failOrSkipped": strings.Join(append(skippedSteps, failedSteps...), ","),
		"result":        result,
	})

	if result == "success" {
		slog.Info(resultMessage)
	} else {
		slog.Error(resultMessage)
		os.Exit(1)
	}
}

func initFunc() {
	// Log as JSON instead of the default ASCII formatter.
	logger := logrus.New()
	if os.Getenv("LOG_FORMAT") == "JSON" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	} else {
		logger.SetFormatter(&logging.TerrascaleFormatter{
			DisableColors: os.Getenv("LOG_DISABLE_COLORS") == "true",
		})
	}
	logger.SetReportCaller(true)
	log = logrus.NewEntry(logger)

	deployment = config.Deployment{}

	var err error

	fs = afero.NewOsFs()

	deployment.Config, err = config.GetConfig()

	if err != nil {
		log.WithError(err).Fatal(err.Error())
	}

	// Only log the warning severity or above.
	lvl, err := logrus.ParseLevel(deployment.Config.LogLevel)

	if err == nil {
		logger.SetLevel(lvl)
	}

	log = logger.WithFields(logrus.Fields{
		"accountID":      deployment.Config.AccountID,
		"deploymentRing": deployment.Config.DeploymentRing,
		//"credsID":                       deployment.Config.CredsID,
		//"csp":                           deployment.Config.CSP, // TODO(config:logging): allow additional logging fields to be passed in
		"project":                   deployment.Config.Project,
		"terrascaleTargetAccountID": deployment.Config.TerrascaleTargetAccountID,
		"environment":               deployment.Config.Environment,
		"namespace":                 deployment.Config.Namespace,
		//"regionGroup":               deployment.Config.TerrascaleRegionGroup, // TODO(config:logging): allow additional logging fields to be passed in
	})

	//// read deployment artifact version string from version.json first, if it exists
	//deployment.DeployMetadata, err = config.GetVersionJSON(log, fs, "version.json")
	//if err != nil {
	//	// defer to the VERSION environment variable instead
	//	deployment.Config.Version = deployment.DeployMetadata.Version
	//}

	if len(deployment.Config.Version) == 0 {
		log.Warn("No version.json or VERSION environment variable specified")
		deployment.Config.Version = ""
	}

	log = log.WithFields(logrus.Fields{
		"version": deployment.Config.Version,
	})

	j, _ := json.MarshalIndent(deployment.Config, "", "    ")

	log.Infof("Parsed configuration: %s", string(j))

	log = log.WithFields(logrus.Fields{
		"uniqueExternalExecutionID": deployment.Config.UniqueExternalExecutionID,
	})

	// init tracker last to ensure log configuration is set correctly
	tracker = tracks.DirectoryBasedTracker{
		Log: log,
		Fs:  fs,
	}

	// prepare basic parameters for executing terraform version
	// disable checkpoints since we just want to print the version string alone
	tfOptions := &terraform.Options{
		TerraformDir: ".",
		EnvVars: map[string]string{
			"CHECKPOINT_DISABLE": "true",
		},
		Logger:             logger.WithField("terraform", "version"),
		NoColor:            true,
		MaxRetries:         1,
		TimeBetweenRetries: 0,
	}

	terraformer := &terraform.Terraform{}
	resp, err := terraformer.Version(tfOptions)
	if err != nil {
		tfOptions.Logger.WithError(err).Error("Error running terraform version")
	} else {
		tfOptions.Logger.Info("Binary: ", resp)
	}
}
