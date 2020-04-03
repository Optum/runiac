package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.optum.com/healthcarecloud/terrascale/pkg/auth"
	"github.optum.com/healthcarecloud/terrascale/pkg/cloudaccountdeployment"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"github.optum.com/healthcarecloud/terrascale/pkg/logging"
	"github.optum.com/healthcarecloud/terrascale/pkg/params"
	"github.optum.com/healthcarecloud/terrascale/pkg/steps"
	"github.optum.com/healthcarecloud/terrascale/pkg/tracks"
	"os"
	"strings"
)

var fs afero.Fs
var tracker tracks.Tracker
var deployment config.Deployment
var log *logrus.Entry

func main() {
	initFunc()

	log.Debugf("Beginning AWS Account Deployment: %s with %s CREDS_ID...", deployment.Config.AccountID, deployment.Config.CredsID)

	log.Debug("Executing tracks...")

	// assume terraform for all steps
	output := tracker.ExecuteTracks(steps.TerraformOnlyStepperFactory{}, deployment.Config)

	log.Debug("Completed executing tracks...")

	trackCount := len(output.Tracks)
	failedSteps := []string{}
	skippedSteps := []string{}
	failedDestroySteps := []string{}
	stepCount := 0
	executedStepCount := 0
	failedTestCount := 0

	for _, t := range output.Tracks {
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
		executedStepCount-failedStepCount, stepCount, failedTestCount, trackCount)

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
	}

	slog := log.WithFields(logrus.Fields{
		"type":   "summary",
		"result": result,
	})

	resultMessage = strings.Title(result) + ".  " + resultMessage

	if result == "success" {
		slog.Info()
	} else {
		slog.Error(resultMessage)
	}
}

func initFunc() {
	// Log as JSON instead of the default ASCII formatter.
	logger := logrus.New()
	if os.Getenv("LOG_FORMAT") == "gaia" {
		logger.SetFormatter(&logging.GaiaFormatter{
			DisableColors: os.Getenv("LOG_DISABLE_COLORS") == "true",
		})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{})
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
		"accountID":               deployment.Config.AccountID,
		"deploymentRing":          deployment.Config.DeploymentRing,
		"credsID":                 deployment.Config.CredsID,
		"csp":                     deployment.Config.CSP,
		"stage":                   deployment.Config.Stage,
		"gaiaTargetAccountID":     deployment.Config.GaiaTargetAccountID,
		"gaiaRingDeploymentID":    deployment.Config.GaiaRingDeploymentID,
		"gaiaReleaseDeploymentID": deployment.Config.GaiaReleaseDeploymentID,
		"environment":             deployment.Config.Environment,
		"namespace":               deployment.Config.Namespace,
		"regionGroup":             deployment.Config.GaiaRegionGroup,
		"lpclagg":                 deployment.Config.GaiaRingDeploymentID,
		"lpcltype":                "gaia",
	})

	deployment.DeployMetadata, err = config.GetVersionJSON(log, fs, "version.json")
	deployment.Config.Version = deployment.DeployMetadata.Version

	if err != nil {
		log.WithError(err).Fatal(err.Error())
	}

	log = log.WithFields(logrus.Fields{
		"version": deployment.DeployMetadata.Version,
	})

	deployment.Config.FargateTaskID, err = config.GetRunningFargateTaskID(deployment.Config.Environment)

	if err != nil {
		log.WithError(err).Fatal(err.Error())
	}

	j, _ := json.Marshal(deployment)

	log.Infof("Parsed configuration: %s", string(j))

	log = log.WithFields(logrus.Fields{
		"fargateTaskID": deployment.Config.FargateTaskID,
	})

	// init tracker last to ensure log configuration is set correctly
	tracker = tracks.DirectoryBasedTracker{
		Log: log,
		Fs:  fs,
	}

	deployment.Config.Authenticator = &auth.SDKAuthenticator{
		Logger:              log,
		BedrockCommonRegion: deployment.Config.CommonRegion,
		AzuCredCache:        make(map[string]*auth.AZUCredentials),
	}

	cloudaccountdeployment.InvokeLambdaFunc = cloudaccountdeployment.InvokeLambdaSDK
	cloudaccountdeployment.Auth = deployment.Config.Authenticator
	cloudaccountdeployment.UpdateStatusLambda = deployment.Config.UpdateStatusLambda

	paramSession, err := deployment.Config.Authenticator.GetPlatformParametersSession(log)

	if err == nil {
		deployment.Config.StepParameters = &params.AWSParamStore{
			Ssmapi: ssm.New(paramSession),
		}
	} else {
		log.WithError(err).Fatal("Failed to initialize Param Store")
	}

}
