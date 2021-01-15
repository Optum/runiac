package steps

import (
	"errors"
	"fmt"
	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.optum.com/healthcarecloud/terrascale/pkg/cloudaccountdeployment"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"github.optum.com/healthcarecloud/terrascale/pkg/terraform"
	plugins_terraform "github.optum.com/healthcarecloud/terrascale/plugins/terraform"
	"path/filepath"
	"strings"
)

type ExecutionConfig struct {
	Stepper                   Stepper
	RegionDeployType          RegionDeployType
	Region                    string `json:"region"`
	Logger                    *logrus.Entry
	Fs                        afero.Fs
	UniqueExternalExecutionID string
	RegionGroupRegions        []string
	TerrascaleTargetAccountID string
	RegionGroup               string
	PrimaryRegion             string
	Dir                       string
	TFBackend                 plugins_terraform.TerraformBackend
	Environment               string `json:"environment"`
	AppVersion                string `json:"app_version"`
	//CredsID                   string `json:"creds_id"`
	AccountID string `json:"account_id"`
	//AccountOwnerID   string `json:"account_owner_msid"`
	MaxRetries       int
	MaxTestRetries   int
	CoreAccounts     map[string]config.Account
	RegionGroups     config.RegionGroupsMap
	Namespace        string
	CommonRegion     string
	StepName         string
	StepID           string
	DeploymentRing   string
	Project          string
	TrackName        string
	DryRun           bool
	SelfDestroy      bool
	TerrascaleConfig TerrascaleConfig

	DefaultStepOutputVariables map[string]map[string]string // Previous step output variables are available in this map. K=StepName,V=map[VarName:VarVal]
	OptionalStepParams         map[string]string
	RequiredStepParams         map[string]interface{}
}

var terraformer terraform.Terraformer = terraform.Terraform{}

func (exec ExecutionConfig) GetCredentialEnvVars() (map[string]string, error) {
	creds := map[string]string{}
	return creds, nil
}

func (exec ExecutionConfig) GetTerraformCLIVars() map[string]interface{} {
	vars := map[string]interface{}{
		"environment": exec.Environment,
		"account_id":  exec.AccountID,
		"region":      exec.Region,
	}

	return vars
}

func (exec ExecutionConfig) GetTerraformEnvVars() map[string]string {
	output := exec.OptionalStepParams

	if exec.Namespace != "" {
		output["namespace"] = exec.Namespace
	}

	if exec.AppVersion != "" {
		output["app_version"] = exec.AppVersion
	}

	// set core accounts
	coreAccountsCount := len(exec.CoreAccounts)
	if exec.CoreAccounts != nil && coreAccountsCount > 0 {
		coreAccounts := "{"

		i := 0
		for k, v := range exec.CoreAccounts {
			coreAccounts += fmt.Sprintf(`"%s":"%s"`, k, v.ID)

			if i < coreAccountsCount-1 {
				coreAccounts += ","
			}
			i++
		}

		coreAccounts += "}"

		output["core_account_ids_map"] = coreAccounts
	}
	// output["account_owner_msid"] = exec.AccountOwnerID
	// output["creds_id"] = exec.CredsID

	return output
}

func NewExecution(s Step, logger *logrus.Entry, fs afero.Fs, regionDeployType RegionDeployType, region string, defaultStepOutputVariables map[string]map[string]string) ExecutionConfig {
	return ExecutionConfig{
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
		TerrascaleConfig:           s.TerrascaleConfig,
		SelfDestroy:                s.DeployConfig.SelfDestroy,
		Logger: logger.WithFields(logrus.Fields{
			"step":            s.Name,
			"stepProgression": s.ProgressionLevel,
		}),
	}
}

func ExecuteStep(stepper Stepper, exec ExecutionConfig) StepOutput {
	output := stepper.ExecuteStep(exec)
	postStep(exec, output)
	return output
}

func ExecuteStepDestroy(stepper Stepper, exec ExecutionConfig) StepOutput {
	return stepper.ExecuteStepDestroy(exec)
}

func ExecuteStepTests(stepper Stepper, exec ExecutionConfig) StepTestOutput {
	output := stepper.ExecuteStepTests(exec)
	postStepTest(exec, output)
	return output
}

func (s Step) InitExecution(logger *logrus.Entry, fs afero.Fs,
	regionDeployType RegionDeployType, region string,
	defaultStepOutputVariables map[string]map[string]string) (
	ExecutionConfig, error) {
	exec := NewExecution(s, logger, fs, regionDeployType, region, defaultStepOutputVariables)

	// set and create execution directory to enable safe concurrency
	if exec.RegionDeployType == RegionalRegionDeployType {
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

	// always ensure we have correct primary region set based on terraform provider csp setting
	//providerTypeToCSP := map[TFProviderType]string{
	//	AWSProvider:     "AWS",
	//	AzurermProvider: "AZU",
	//}

	// TODO(config:region):  allow this to be set via external configuration (terrascale.yml)
	//providerCSP := ""
	//if csp, ok := providerTypeToCSP[provider.Type]; ok {
	//	providerCSP = csp
	//}
	//
	//if providerCSP != "" {
	//	exec.PrimaryRegion = s.DeployConfig.GetPrimaryRegionByCSP(providerCSP)
	//
	//	if exec.RegionDeployType == PrimaryRegionDeployType {
	//		exec.Region = exec.PrimaryRegion
	//
	//		exec.Logger = exec.Logger.WithField("region", exec.Region)
	//		exec.Logger.Infof("Set region to %s based on %s provider's primary region", exec.Region, providerCSP)
	//	}
	//}

	//if provider.AccountOverridden {
	//	exec.AccountID = provider.AssumeRoleAccount.ID
	//	exec.CredsID = provider.AssumeRoleAccount.CredsID
	//	exec.AccountOwnerID = provider.AssumeRoleAccount.AccountOwnerLabel
	//
	//	// if no account was originally targeted in this run, use this specific step's "AccountOveridden" account id
	//	if exec.TerrascaleTargetAccountID == "" {
	//		exec.TerrascaleTargetAccountID = exec.AccountID
	//	}
	//
	//	exec.Logger.Infof("Overriding account to %v/%v based on provider.tf", exec.AccountID, exec.CredsID)
	//}

	exec.Logger = exec.Logger.WithFields(logrus.Fields{
		"accountID": exec.AccountID,
	})

	var params = map[string]string{}

	// translate custom type to map type for terraformer to parse correctly
	var rgs map[string]map[string][]string = s.DeployConfig.RegionGroups

	// Add Terrascale variables to step params
	params["terrascale_target_account_id"] = exec.TerrascaleTargetAccountID
	params["terrascale_deployment_ring"] = exec.DeploymentRing
	params["terrascale_project"] = strings.ToLower(exec.Project)
	params["terrascale_track"] = strings.ToLower(exec.TrackName)
	params["terrascale_step"] = strings.ToLower(exec.StepName)
	params["terrascale_region_deploy_type"] = strings.ToLower(exec.RegionDeployType.String())
	params["terrascale_region_group"] = strings.ToLower(exec.RegionGroup)
	params["terrascale_region_group_regions"] = strings.Replace(terraformer.OutputToString(s.DeployConfig.RegionalRegions), " ", ",", -1)
	params["terrascale_primary_region"] = exec.PrimaryRegion
	params["terrascale_region_groups"] = terraformer.OutputToString(rgs)

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

	exec.TFBackend = plugins_terraform.GetBackendConfig(exec, plugins_terraform.ParseTFBackend)
	exec.Stepper = plugins_terraform.TerraformStepper{}

	return exec, nil
}

func postStep(exec ExecutionConfig, output StepOutput) {
	if output.Err != nil {
		cloudaccountdeployment.RecordStepFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, output.Err)
	} else if output.Status == Fail {
		cloudaccountdeployment.RecordStepFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, errors.New("step recorded failure with no error thrown"))
	} else if output.Status == Unstable {
		cloudaccountdeployment.RecordStepFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, errors.New("step recorded unstable with no error thrown"))
	} else {
		cloudaccountdeployment.RecordStepSuccess(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions)
	}
}

func postStepTest(exec ExecutionConfig, output StepTestOutput) {
	if output.Err != nil {
		cloudaccountdeployment.RecordStepTestFail(exec.Logger, "", exec.TrackName, exec.StepName, exec.RegionDeployType.String(), exec.Region, exec.UniqueExternalExecutionID, exec.Project, exec.RegionGroupRegions, output.Err)
	}
}
