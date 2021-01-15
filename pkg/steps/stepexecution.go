package steps

import (
	"errors"
	"fmt"
	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.optum.com/healthcarecloud/terrascale/pkg/cloudaccountdeployment"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"github.optum.com/healthcarecloud/terrascale/plugins/terraform/pkg/terraform"
	"path/filepath"
	"strings"
)

func NewExecution(s config.Step, logger *logrus.Entry, fs afero.Fs, regionDeployType config.RegionDeployType, region string, defaultStepOutputVariables map[string]map[string]string) config.StepExecution {
	return config.StepExecution{
		RegionDeployType:           regionDeployType,
		Region:                     region,
		Fs:                         fs,
		TerrascaleTargetAccountID:  s.DeployConfig.TerrascaleTargetAccountID,
		RegionGroup:                s.DeployConfig.TerrascaleRegionGroup,
		DefaultStepOutputVariables: defaultStepOutputVariables,
		Environment:                s.DeployConfig.Environment,
		AppVersion:                 s.DeployConfig.Version,
		AccountID:                  s.DeployConfig.AccountID,
		CoreAccounts:               s.DeployConfig.CoreAccounts,
		StepName:                   s.Name,
		StepID:                     s.ID,
		Namespace:                  s.DeployConfig.Namespace,
		Dir:                        s.Dir,
		DeploymentRing:             s.DeployConfig.DeploymentRing,
		DryRun:                     s.DeployConfig.DryRun,
		MaxRetries:                 s.DeployConfig.MaxRetries,
		MaxTestRetries:             s.DeployConfig.MaxTestRetries,
		Project:                    s.DeployConfig.Project,
		TrackName:                  s.TrackName,
		RegionGroupRegions:         s.DeployConfig.RegionalRegions,
		UniqueExternalExecutionID:  s.DeployConfig.UniqueExternalExecutionID,
		RegionGroups:               s.DeployConfig.RegionGroups,
		SelfDestroy:                s.DeployConfig.SelfDestroy,
		Logger: logger.WithFields(logrus.Fields{
			"step":            s.Name,
			"stepProgression": s.ProgressionLevel,
		}),
	}
}

func ExecuteStep(stepper config.Stepper, exec config.StepExecution) config.StepOutput {

	// Check if the step is filtered in the configuration // TODO: step configuration override
	//inRegions := exec.TerrascaleConfig.ExecuteWhen.RegionIn
	//if len(inRegions) > 0 && !contains(inRegions, exec.Region) {
	//	exec.Logger.Warn("Skipping execution. Region is not included in the execute_when.region_in configuration")
	//	return steps.StepOutput{
	//		Status:           steps.Na,
	//		RegionDeployType: exec.RegionDeployType,
	//		Region:           exec.Region,
	//		StepName:         exec.StepName,
	//		StreamOutput:     "",
	//		Err:              nil,
	//		OutputVariables:  nil,
	//	}
	//}

	exec.Logger.Debugf("%v", exec.RequiredStepParams)
	exec.Logger.Debugf("%v", exec.OptionalStepParams)

	output := stepper.ExecuteStep(exec)
	postStep(exec, output)
	return output
}

func ExecuteStepDestroy(stepper config.Stepper, exec config.StepExecution) config.StepOutput {
	return stepper.ExecuteStepDestroy(exec)
}

func ExecuteStepTests(stepper config.Stepper, exec config.StepExecution) config.StepTestOutput {
	output := stepper.ExecuteStepTests(exec)
	postStepTest(exec, output)
	return output
}

func InitExecution(s config.Step, logger *logrus.Entry, fs afero.Fs,
	regionDeployType config.RegionDeployType, region string,
	defaultStepOutputVariables map[string]map[string]string) (
	config.StepExecution, error) {
	exec := NewExecution(s, logger, fs, regionDeployType, region, defaultStepOutputVariables)

	// set and create execution directory to enable safe concurrency
	if exec.RegionDeployType == config.RegionalRegionDeployType {
		regionalDir := filepath.Join(s.Dir, "regional")
		execRegionalDir := filepath.Join(s.Dir, fmt.Sprintf("regional-%s", exec.Region))
		err := exec.Fs.MkdirAll(execRegionalDir, 0700)

		if err != nil {
			exec.Logger.WithError(err).Error(err)
			return exec, err
		}

		exec.Logger.Infof("Copying %s regional to %s", exec.Region, execRegionalDir)

		err = copy.Copy(regionalDir, execRegionalDir)

		if err != nil {
			exec.Logger.WithError(err).Error(err)
			return exec, err
		}

		exec.Dir = execRegionalDir
	}

	accounts := map[string]config.Account{
		"terrascale_target_account_id": {
			ID: exec.TerrascaleTargetAccountID,
		},
	}
	for k, v := range exec.CoreAccounts {
		accounts[k] = v
	}

	exec.Logger = exec.Logger.WithFields(logrus.Fields{
		"accountID": exec.AccountID,
	})

	var params = map[string]string{}

	// Add Terrascale variables to step params
	params["terrascale_target_account_id"] = exec.TerrascaleTargetAccountID
	params["terrascale_deployment_ring"] = exec.DeploymentRing
	params["terrascale_project"] = strings.ToLower(exec.Project)
	params["terrascale_track"] = strings.ToLower(exec.TrackName)
	params["terrascale_step"] = strings.ToLower(exec.StepName)
	params["terrascale_region_deploy_type"] = strings.ToLower(exec.RegionDeployType.String())
	params["terrascale_region_group"] = strings.ToLower(exec.RegionGroup)
	//params["terrascale_region_group_regions"] = strings.Replace(terraformer.OutputToString(s.DeployConfig.RegionalRegions), " ", ",", -1) // TODO
	params["terrascale_primary_region"] = exec.PrimaryRegion
	//params["terrascale_region_groups"] = terraformer.OutputToString(rgs) // TODO

	// TODO: pre-step plugin for integrating "just-in-time" variables from external source
	exec.Logger.Debugf("output variables: %s", KeysStringMap(exec.DefaultStepOutputVariables))

	// Add previous step outputs from the track into stepParams
	stepParams := AppendToStepParams(params, exec.DefaultStepOutputVariables)

	// if step has an output, add here (primarily for tests)
	// TODO: find a better way to handle this that doesn't rely on re-calling this method for tests
	if s.Output.OutputVariables != nil {
		for k, v := range s.Output.OutputVariables {
			params[k] = terraform.OutputToString(v)
		}
	} else {
		cloudaccountdeployment.RecordStepStart(exec.Logger, exec.AccountID, exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.DryRun, "", exec.AppVersion, s.DeployConfig.UniqueExternalExecutionID, "", "", exec.Project, s.DeployConfig.RegionalRegions)
	}

	exec.OptionalStepParams = stepParams

	return exec, nil
}

func postStep(exec config.StepExecution, output config.StepOutput) {
	if output.Err != nil {
		cloudaccountdeployment.RecordStepFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, output.Err)
	} else if output.Status == config.Fail {
		cloudaccountdeployment.RecordStepFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, errors.New("step recorded failure with no error thrown"))
	} else if output.Status == config.Unstable {
		cloudaccountdeployment.RecordStepFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, errors.New("step recorded unstable with no error thrown"))
	} else {
		cloudaccountdeployment.RecordStepSuccess(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions)
	}
}

func postStepTest(exec config.StepExecution, output config.StepTestOutput) {
	if output.Err != nil {
		cloudaccountdeployment.RecordStepTestFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, output.Err)
	}
}
