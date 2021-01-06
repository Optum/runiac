package cloudaccountdeployment

import (
	"encoding/json"
	"fmt"
	"github.optum.com/healthcarecloud/terrascale/pkg/auth"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/sirupsen/logrus"
)

type DeployPhase int

const (
	PreDeploy DeployPhase = iota
	PostDeploy
	RegionalPostDeploy
)

func (d DeployPhase) String() string {
	return [...]string{"PREDEPLOY", "POSTDEPLOY", "REGIONALPOSTDEPLOY"}[d]
}

type DeployResult int

const (
	InProgress DeployResult = iota
	Success
	Fail
	Unstable
)

func (d DeployResult) String() string {
	return [...]string{"INPROGRESS", "SUCCESS", "FAIL", "UNSTABLE"}[d]
}

type UpdateStatusPayload struct {
	Product             string   `json:"product"`
	AccountID           string   `json:"account_id"`
	CSP                 string   `json:"csp"`
	DeploymentPhase     string   `json:"deployment_phase"`
	Version             string   `json:"version"`
	Result              string   `json:"result"`
	ResultMessage       string   `json:"result_message"`
	Tool                string   `json:"tool"`
	AccountDeploymentID string   `json:"account_deployment_id"`
	RingDeploymentID    string   `json:"ring_deployment_id"`
	ReleaseDeploymentID string   `json:"release_deployment_id"`
	Stage               string   `json:"stage"`
	Track               string   `json:"track"`
	Step                string   `json:"step"`
	TargetRegions       []string `json:"targeted_regions"`
	PrimaryRegion       string   `json:"primary_region"`
}

type UpdateRegionalStatusPayload struct {
	AccountStepDeploymentID string            `json:"account_step_deployment_id"`
	FailedRegions           []string          `json:"failed_regions"`
	DeploymentPhase         string            `json:"deployment_phase"`
	CSP                     string            `json:"csp"`
	Result                  string            `json:"result"`
	ResultMessage           string            `json:"result_message"`
	TargetRegions           []string          `json:"-"`
	Executions              []ExecutionResult `json:"-"`
}

type ExecutionResult struct {
	Result                  DeployResult
	Region                  string
	RegionDeployType        string
	AccountStepDeploymentID string
	CSP                     string
	TargetRegions           []string
}

var StepDeployments = map[string]ExecutionResult{}
var Cfg, _ = config.GetConfig()

func RecordStepStart(logger *logrus.Entry, accountID string, track string, step string, regionDeployType string, region string, dryRun bool, csp string, version string, executionID string, stepFunctionName string, codePipelineExecutionID string, stage string, terrascaleTargetRegions []string) {
	deployPhase := PreDeploy
	result := InProgress
	resultMessage := ""

	// only record start of primary region
	if regionDeployType != "primary" {
		return
	}

	if dryRun {
		logger.Info("Skipping updateStatus during DryRun")
		return
	}

	logger.Debug("Updating deployment status...")

	p := UpdateStatusPayload{
		Product:             stage,
		AccountID:           accountID,
		CSP:                 csp,
		DeploymentPhase:     deployPhase.String(),
		Version:             version,
		Result:              result.String(),
		ResultMessage:       resultMessage,
		Tool:                "StepFn",
		AccountDeploymentID: executionID,
		RingDeploymentID:    stepFunctionName,
		ReleaseDeploymentID: codePipelineExecutionID,
		Stage:               stage,
		Step:                step,
		Track:               track,
		PrimaryRegion:       region,
		TargetRegions:       terrascaleTargetRegions,
	}

	if Cfg.ReporterDynamodb {
		_ = InvokeLambdaFunc(logger, p)
	}
}

func RecordStepSuccess(logger *logrus.Entry, csp string, track string, step string, regionDeployType string, region string, executionID string, stage string, terrascaleTargetRegions []string) {
	result := Success
	//resultMessage := "Success"

	StepDeployments[fmt.Sprintf("#%s#%s#%s#%s", track, step, regionDeployType, region)] = ExecutionResult{
		Result:                  result,
		Region:                  region,
		RegionDeployType:        regionDeployType,
		AccountStepDeploymentID: fmt.Sprintf("%s#%s#%s#%s", executionID, stage, track, step),
		CSP:                     csp,
		TargetRegions:           terrascaleTargetRegions,
	}
}

func RecordStepFail(logger *logrus.Entry, csp string, track string, step string, regionDeployType string, region string, executionID string, stage string, terrascaleTargetRegions []string, err error) {
	result := Fail
	//resultMessage := ""

	StepDeployments[fmt.Sprintf("#%s#%s#%s#%s", track, step, regionDeployType, region)] = ExecutionResult{
		Result:                  result,
		Region:                  region,
		RegionDeployType:        regionDeployType,
		AccountStepDeploymentID: fmt.Sprintf("%s#%s#%s#%s", executionID, stage, track, step),
		CSP:                     csp,
		TargetRegions:           terrascaleTargetRegions,
	}
}

func RecordStepTestFail(logger *logrus.Entry, csp string, track string, step string, regionDeployType string, region string, executionID string, stage string, terrascaleTargetRegions []string, err error) {
	result := Unstable

	StepDeployments[fmt.Sprintf("#%s#%s#%s#%s", track, step, regionDeployType, region)] = ExecutionResult{
		Result:                  result,
		Region:                  region,
		RegionDeployType:        regionDeployType,
		AccountStepDeploymentID: fmt.Sprintf("%s#%s#%s#%s", executionID, stage, track, step),
		CSP:                     csp,
		TargetRegions:           terrascaleTargetRegions,
	}
}

// Flush track will record a track's regional deployments
func FlushTrack(logger *logrus.Entry, track string) (steps map[string]*UpdateRegionalStatusPayload, err error) {
	steps = map[string]*UpdateRegionalStatusPayload{}
	flushedSteps := []string{}

	if len(StepDeployments) == 0 {
		logger.Warnf("FlushTrack: No steps to flush for track")
	}

	for k, v := range StepDeployments {
		if !strings.HasPrefix(k, "#"+track+"#") {
			continue
		}

		if steps[v.AccountStepDeploymentID] == nil {
			steps[v.AccountStepDeploymentID] = &UpdateRegionalStatusPayload{
				AccountStepDeploymentID: v.AccountStepDeploymentID,
				FailedRegions:           []string{},
				TargetRegions:           v.TargetRegions,
				DeploymentPhase:         RegionalPostDeploy.String(),
				CSP:                     v.CSP,
				Executions:              []ExecutionResult{},
			}
		}

		steps[v.AccountStepDeploymentID].Executions = append(steps[v.AccountStepDeploymentID].Executions, v)

		if v.Result != Success && v.Result != InProgress {
			steps[v.AccountStepDeploymentID].FailedRegions = append(steps[v.AccountStepDeploymentID].FailedRegions, fmt.Sprintf("%s/%s", v.RegionDeployType, v.Region))
		}

		flushedSteps = append(flushedSteps, k)
	}

	for stepID, v := range steps {
		failedExecutionCount := len(v.FailedRegions)
		if failedExecutionCount >= len(v.TargetRegions) {
			v.Result = Fail.String()
		} else if failedExecutionCount > 0 {
			v.Result = Unstable.String()
		} else {
			v.Result = Success.String()
		}

		includeRegional := false
		includePrimary := false
		primaryRegion := ""
		primaryFailCount := 0
		regionalFailCount := 0
		failures := []string{}
		for _, execution := range v.Executions {
			if execution.RegionDeployType == "regional" {
				includeRegional = true
				if execution.Result == Fail || execution.Result == Unstable {
					regionalFailCount++
					failures = append(failures, fmt.Sprintf("%s/%s", execution.RegionDeployType, execution.Region))
				}
			}

			if execution.RegionDeployType == "primary" {
				includePrimary = true
				primaryRegion = execution.Region

				if execution.Result == Fail || execution.Result == Unstable {
					primaryFailCount++
					failures = append(failures, fmt.Sprintf("%s/%s", execution.RegionDeployType, execution.Region))
				}
			}
		}

		v.ResultMessage += fmt.Sprintf("%s:", v.Result)

		if includePrimary {
			v.ResultMessage += fmt.Sprintf(" Primary resources applied to %s.", primaryRegion)
		}

		if includeRegional {
			v.ResultMessage += fmt.Sprintf("  Regional resources applied to %s.", strings.Join(v.TargetRegions, ", "))
		} else {
			v.ResultMessage += "  No regional resources for step."
		}

		if len(failures) > 0 {
			v.ResultMessage += fmt.Sprintf("  Failed executions: %s", strings.Join(failures, ", "))
		}

		logger.Infof("%s: %s", stepID, v.ResultMessage)

		if Cfg.ReporterDynamodb {
			_ = InvokeLambdaFunc(logger.WithField("accountStepDeployID", v.AccountStepDeploymentID), *v)
		}
	}

	// reset step deployments
	for _, flushedStep := range flushedSteps {
		delete(StepDeployments, flushedStep)
	}

	return steps, err
}

type InvokeLambda func(logger *logrus.Entry, p interface{}) map[string]interface{}

var InvokeLambdaFunc InvokeLambda

var Auth auth.Authenticator
var UpdateStatusLambda string

func InvokeLambdaSDK(logger *logrus.Entry, p interface{}) map[string]interface{} {
	platformSession := Auth.GetPlatformSession()

	svc := lambda.New(platformSession)

	payload, err := json.Marshal(p)
	logger.Debugf("Calling deployment status update with payload: %s\n", payload)

	if err != nil {
		logger.WithError(err).Error("Json Marshalling Error")
	}

	input := &lambda.InvokeInput{
		FunctionName:   aws.String(UpdateStatusLambda),
		InvocationType: aws.String("RequestResponse"),
		LogType:        aws.String("Tail"),
		Payload:        payload,
	}

	resp, err := svc.Invoke(input)
	if err != nil {
		logger.WithError(err).Error("Error invoking update status lambda")
		return nil
	}

	if resp.FunctionError != nil {
		logger.WithField("requestPayload", payload).WithError(fmt.Errorf(string(resp.Payload))).Error("Lambda execution error occurred")
	}

	var m map[string]interface{}
	json.Unmarshal(resp.Payload, &m)

	return m
}
