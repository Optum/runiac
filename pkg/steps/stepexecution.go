package steps

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.optum.com/healthcarecloud/terrascale/pkg/cloudaccountdeployment"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"github.optum.com/healthcarecloud/terrascale/pkg/retry"
	"github.optum.com/healthcarecloud/terrascale/pkg/shell"
	"github.optum.com/healthcarecloud/terrascale/pkg/terraform"
)

type TerraformStepper struct{}

type ExecutionConfig struct {
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
	TFBackend                 TerraformBackend
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
		Logger: logger.WithFields(logrus.Fields{
			"step":            s.Name,
			"stepProgression": s.ProgressionLevel,
		}),
	}
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

	exec.TFBackend = GetBackendConfig(exec, ParseTFBackend)

	// Handle any override files in the step and deployment ring. Deploy
	// overrides will always copied over and they destroy overrides will be
	// copied over only on Self-Destroy.
	HandleDeployOverrides(exec.Logger, exec.Dir, exec.DeploymentRing)

	if s.DeployConfig.SelfDestroy {
		HandleDestroyOverrides(exec.Logger, exec.Dir, exec.DeploymentRing)
	}

	return exec, nil
}

func handleOverride(logger *logrus.Entry, execDir string, fileName string) {
	src := filepath.Join(execDir, "override", fileName)
	dst := filepath.Join(execDir, fileName)

	logger.Infof("Attempting to copy %s to %s", src, dst)

	err := CopyFile(src, dst)

	if err != nil && !os.IsNotExist(err) {
		logger.WithError(err).Errorf(
			"Overrides were not successfully set targeting %s", fileName)
	}
}

// HandleDeployOverrides copy deploy override configurations into the
// execution working directory
func HandleDeployOverrides(logger *logrus.Entry, execDir string,
	deploymentRing string) {
	overrideFile := "override.tf"
	ringOverrideFile := fmt.Sprintf("ring_%s_override.tf",
		strings.ToLower(deploymentRing))

	handleOverride(logger, execDir, overrideFile)
	handleOverride(logger, execDir, ringOverrideFile)
}

// HandleDestroyOverrides copy destroy override configurations into the
// execution working directory
func HandleDestroyOverrides(logger *logrus.Entry, execDir string,
	deploymentRing string) {
	destroyOverrideFile := "destroy_override.tf"
	destroyRingOverrideFile := fmt.Sprintf("destroy_ring_%s_override.tf",
		strings.ToLower(deploymentRing))

	handleOverride(logger, execDir, destroyOverrideFile)
	handleOverride(logger, execDir, destroyRingOverrideFile)
}

// ExecuteStepDestroy destroys a step
func (stepper TerraformStepper) ExecuteStepDestroy(exec ExecutionConfig) StepOutput {
	return executeTerraformInDir(exec, true)
}

// ExecuteStep deploys a step
func (stepper TerraformStepper) ExecuteStep(exec ExecutionConfig) StepOutput {
	output := executeTerraformInDir(exec, false)
	postStep(exec, output)

	return output
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// ExecuteStepTests executes the tests for a step
func (stepper TerraformStepper) ExecuteStepTests(exec ExecutionConfig) (output StepTestOutput) {
	envVars := map[string]string{}

	for k, v := range exec.GetTerraformEnvVars() {
		envVars[fmt.Sprintf("TF_VAR_%s", k)] = v
	}

	for k, v := range exec.GetTerraformCLIVars() {
		envVars[fmt.Sprintf("TF_VAR_%s", k)] = fmt.Sprintf("%v", v)
	}

	// Grab initial creds for the deployment
	creds, err := exec.GetCredentialEnvVars()

	if err != nil {
		exec.Logger.WithError(err).Error("unable to retrieve credentials for step tests")
		output.Err = err
		return output
	}

	// set credential environment variables
	for k, v := range creds {
		envVars[k] = v
	}

	testDir := fmt.Sprintf("%s/tests", exec.Dir)

	// ensure output directory exists for test reporting
	outputDir := filepath.Join("/", "output", "junit")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, os.ModePerm)

		if err != nil {
			exec.Logger.WithError(err).Warn("Failed to create output directory for test results")
		}
	}

	_ = retry.DoWithRetry(fmt.Sprintf("execute tests: %s", testDir), exec.MaxTestRetries, 20*time.Second, exec.Logger, func(retryCount int) error {
		retryLogger := exec.Logger.WithField("retryCount", retryCount)
		stepDeployID := fmt.Sprintf("%s-%s-%s-%s-%s", exec.Project, exec.TrackName, exec.StepName, exec.RegionDeployType, exec.Region)
		cmd := shell.Command{
			Command: "gotestsum",
			//Command:        "/bin/bash",
			Args:           []string{"--format", "standard-verbose", "--junitfile", fmt.Sprintf("/output/junit/%s.xml", stepDeployID), "--raw-command", "--", "test2json", "-p", stepDeployID, "./tests.test", "-test.v"},
			Logger:         retryLogger,
			SensitiveArgs:  false,
			NonInteractive: true,
			Env:            envVars,
			WorkingDir:     testDir,
		}

		output.StreamOutput, output.Err = shell.RunShellCommandAndGetAndStreamOutput(cmd)

		return output.Err
	})

	postStepTest(exec, output)

	return
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

// executeTerraformInDir is a helper function for executing terraform in a specified directory
var executeTerraformInDir = func(exec ExecutionConfig, destroy bool) (output StepOutput) {
	output.RegionDeployType = exec.RegionDeployType
	output.Region = exec.Region
	output.StepName = exec.StepName
	output.Status = Fail // assume failure
	var resp string
	var tfOptions *terraform.Options

	// Check if the step is filtered in the configuration
	inRegions := exec.TerrascaleConfig.ExecuteWhen.RegionIn
	if len(inRegions) > 0 && !contains(inRegions, exec.Region) {
		exec.Logger.Warn("Skipping execution. Region is not included in the execute_when.region_in configuration")
		return StepOutput{
			Status:           Na,
			RegionDeployType: exec.RegionDeployType,
			Region:           exec.Region,
			StepName:         exec.StepName,
			StreamOutput:     "",
			Err:              nil,
			OutputVariables:  nil,
		}
	}

	// terraform init
	tfOptions, output.Err = getCommonTfOptions2(exec)

	if output.Err != nil {
		tfOptions.Logger.WithError(output.Err).Error("unable to retrieve credentials for terraform init")
		return
	}

	tfOptions.BackendConfig = exec.TFBackend.Config
	tfOptions.Logger = tfOptions.Logger.WithField("terraform", "init")
	resp, output.Err = terraformer.Init(tfOptions)

	if output.Err != nil {
		tfOptions.Logger.WithError(output.Err).Error("Error during terraform init")
		return
	}

	// terraform plan
	_ = retry.DoWithRetry("terraform plan and apply", tfOptions.MaxRetries, 10*time.Second, tfOptions.Logger, func(attempt int) error {

		retryLogger := tfOptions.Logger.WithField("retryCount", attempt)

		tfplan := fmt.Sprintf("%s%s%stfplan", exec.StepName, exec.RegionDeployType, exec.Region)

		// terraform plan
		tfOptions, output.Err = getCommonTfOptions2(exec)

		if output.Err != nil {
			tfOptions.Logger.WithError(output.Err).Error("Error running terraform plan")
			return output.Err
		}

		tfOptions.Logger = retryLogger.WithField("terraform", "plan")

		// Set all step parameters as terraform env variables
		for k, v := range exec.GetTerraformEnvVars() {
			tfOptions.Logger.Infof("Adding parameter to TF_VARs: %s", k)
			tfOptions.EnvVars[fmt.Sprintf("TF_VAR_%s", k)] = v
		}

		tfOptions.Vars = exec.GetTerraformCLIVars()

		resp, output.Err = terraformer.Plan(tfOptions, tfplan, destroy)

		if output.Err != nil {
			tfOptions.Logger.WithError(output.Err).Error("Error running terraform plan")
			return output.Err
		}

		// validate terraform plan
		// new options to reset variables
		baseOptions, err := getCommonTfOptions2(exec)

		if err != nil {
			retryLogger.WithError(output.Err).Error("Error retrieving tf options for terraform show")
		}

		baseOptions.Logger = retryLogger.WithField("terraform", "show")
		resp, output.Err = terraformer.Show(baseOptions, tfplan)

		if output.Err != nil {
			baseOptions.Logger.WithError(output.Err).Errorf("Error during terraform show:\n%s", resp)
			return output.Err
		}

		plan := plan{}
		output.Err = json.Unmarshal([]byte(resp), &plan)

		if output.Err != nil {
			tfOptions.Logger.WithError(output.Err).Error("Error unmarshalling terraform show")
			return output.Err
		}
		// aws_cloudtrail.central_logging_trail, aws_cloudtrail, central_logging_trail: [no-op]

		resourceChangesByAction := map[string][]string{}
		for _, c := range plan.ResourceChanges {
			key := fmt.Sprintf("%s", c.Change.Actions)
			if resourceChangesByAction[key] == nil {
				resourceChangesByAction[key] = []string{}
			}

			resourceChangesByAction[key] = append(resourceChangesByAction[key], c.Address)

			tfOptions.Logger.Info(fmt.Sprintf("%s, %s, %s: %s", c.Address, c.Type, c.Name, c.Change.Actions))
		}
		applyChanges := true
		//noChanges := len(resourceChangesByAction["[no-op]"]) == len(plan.ResourceChanges)

		// only run apply on when not dry run and changes exist
		if exec.DryRun {
			tfOptions.Logger.Info("---------- Skipping apply, this is a dry run ---------- ")
			applyChanges = false
		}

		//if noChanges {
		//	tfOptions.Logger.Info("---------- Skipping apply, no changes detected ---------- ")
		//	applyChanges = false
		//}

		if applyChanges {
			// terraform apply
			baseOptions.Logger = retryLogger.WithField("terraform", "apply")
			resp, output.Err = terraformer.Apply(baseOptions, tfplan)

			if output.Err != nil {
				baseOptions.Logger.WithError(output.Err).Error("Error running terraform apply")
				return output.Err
			}
		}

		// parse terraform output
		baseOptions, output.Err = getCommonTfOptions2(exec)

		if output.Err != nil {
			retryLogger.WithError(output.Err).Error("unable to retrieve credentials for terraform output")
			return output.Err
		}

		baseOptions.Logger = retryLogger.WithField("terraform", "output")

		output.OutputVariables, output.Err = terraformer.OutputAll(tfOptions)

		if output.Err != nil {
			baseOptions.Logger.WithError(output.Err).Error("Error running terraform output")
		}

		output.Status = Success

		return nil
	})

	return
}

// GetBackendConfig parses a backend.tf file
// TODO, replace this with a cleaner hcl2json2struct merge where backend.tf configurations take priority over defined defaults here
func GetBackendConfig(exec ExecutionConfig, backendParser TFBackendParser) TerraformBackend {
	declaredBackend := backendParser(exec.Fs, exec.Logger, filepath.Join(exec.Dir, "backend.tf"))

	exec.Logger.Debugf("Parsed Backend Type: %s", declaredBackend.Type)
	exec.Logger.Debugf("Parsed Backend Key: %s", declaredBackend.Key)

	backendConfig := map[string]map[string]interface{}{
		"s3":      {},
		"azurerm": {},
		"gcs":     {},
		"local":   {},
	}

	// if user has decided to set a specific backend type, use that and set default values
	b := backendConfig[declaredBackend.Type.String()]

	// if user has decided to overwrite state file convention in backend.tf, support this override
	if declaredBackend.Key != "" {
		// grab statefile name (base)
		b["key"] = interpolateString(exec, declaredBackend.Key)
	}

	if declaredBackend.S3RoleArn != "" {
		b["role_arn"] = interpolateString(exec, declaredBackend.S3RoleArn)

		exec.Logger.Debugf("Declared S3RoleArn: %s", b["role_arn"])
	}

	if declaredBackend.S3Bucket != "" {
		b["bucket"] = interpolateString(exec, declaredBackend.S3Bucket)

		exec.Logger.Debugf("Declared S3 bucket: %s", b["bucket"])
	}

	if declaredBackend.GCSBucket != "" {
		b["bucket"] = interpolateString(exec, declaredBackend.GCSBucket)

		exec.Logger.Debugf("Declared GCS bucket: %s", b["bucket"])
	}

	if declaredBackend.GCSPrefix != "" {
		b["prefix"] = interpolateString(exec, declaredBackend.GCSPrefix)

		exec.Logger.Debugf("Declared GCS prefix: %s", b["prefix"])
	}

	if declaredBackend.AZUResourceGroupName != "" {
		b["resource_group_name"] = interpolateString(exec, declaredBackend.AZUResourceGroupName)
	}

	if declaredBackend.AZUStorageAccountName != "" {
		b["storage_account_name"] = interpolateString(exec, declaredBackend.AZUStorageAccountName)
	}

	if declaredBackend.AZUStorageAccountName != "" {
		b["storage_account_name"] = interpolateString(exec, declaredBackend.AZUStorageAccountName)
	}

	if declaredBackend.Path != "" {
		b["path"] = interpolateString(exec, declaredBackend.Path)
	}

	declaredBackend.Config = b

	return declaredBackend
}

func interpolateString(exec ExecutionConfig, s string) string {
	if strings.Contains(s, "${var.terrascale_deployment_ring}") {
		s = strings.ReplaceAll(s, "${var.terrascale_deployment_ring}", exec.DeploymentRing)
	}

	if strings.Contains(s, "${var.terrascale_target_account_id}") {
		s = strings.ReplaceAll(s,
			"${var.terrascale_target_account_id}", exec.TerrascaleTargetAccountID)
	}

	if strings.Contains(s, "${var.terrascale_step}") {
		s = strings.ReplaceAll(s,
			"${var.terrascale_step}", exec.StepName)
	}

	if strings.Contains(s, "${var.terrascale_region_deploy_type}") {
		s = strings.ReplaceAll(s,
			"${var.terrascale_region_deploy_type}", exec.RegionDeployType.String())
	}

	if strings.Contains(s, "${var.region}") {
		s = strings.ReplaceAll(s,
			"${var.region}", exec.Region)
	}

	if strings.Contains(s, "${local.namespace-}") {
		namespace_ := exec.Namespace

		if len(namespace_) > 0 {
			namespace_ += "-"
		}

		s = strings.ReplaceAll(s,
			"${local.namespace-}", namespace_)
	}

	if strings.Contains(s, "${var.region}") {
		s = strings.ReplaceAll(s,
			"${var.terrascale_step}", exec.StepName)
	}

	if strings.Contains(s, "${var.environment}") {
		s = strings.ReplaceAll(s,
			"${var.environment}", exec.Environment)
	}

	// Replace all ${var.core_account_ids_map instances.
	// There could be multiple ${var.core_account_ids_map references in the string,
	if strings.Contains(s, "${var.core_account_ids_map") {
		regexForAllCoreAccountIdsMap := regexp.MustCompile(`(?m)\${var\.core_account_ids_map\..*?}`)
		matches := regexForAllCoreAccountIdsMap.FindAllString(s, -1)

		for _, match := range matches {
			// Expected match will look like: ${var.core_account_ids_map.logging_bridge_gcp}
			splitOnCoreAccountIdsMap := strings.Split(match, "${var.core_account_ids_map.")
			if len(splitOnCoreAccountIdsMap) != 2 {
				exec.Logger.Errorf("Error translating core_account_ids_map map for regex match: %s. Unexpected split on core_account_ids_map.", match)
				continue
			}

			coreAccountNameWithClosingCurlyBracket := splitOnCoreAccountIdsMap[1]
			splitOnClosingCurlyBracket := strings.Split(coreAccountNameWithClosingCurlyBracket, "}")
			if len(splitOnClosingCurlyBracket) != 2 {
				exec.Logger.Errorf("Error translating core_account_ids_map map for regex match: %s. Unexpected split on closing }.", match)
				continue
			}

			coreAccountName := splitOnClosingCurlyBracket[0]
			if coreAccount, coreAccountExists := exec.CoreAccounts[coreAccountName]; coreAccountExists {
				s = strings.ReplaceAll(s, match, coreAccount.ID)
			} else {
				exec.Logger.Errorf("Did not find %s in the core accounts map. Core accounts map keys are: %+v", coreAccountName, KeysString(exec.CoreAccounts))
			}
		}
	}

	return s
}

func getCommonTfOptions2(exec ExecutionConfig) (tfOptions *terraform.Options, err error) {
	tfOptions = &terraform.Options{
		TerraformDir:             exec.Dir,
		EnvVars:                  map[string]string{},
		Logger:                   exec.Logger,
		NoColor:                  true,
		RetryableTerraformErrors: map[string]string{".*": "General Terraform error occurred."},
		MaxRetries:               exec.MaxRetries,
		TimeBetweenRetries:       5 * time.Second,
	}

	// Grab initial credentials for the deployment
	creds, err := exec.GetCredentialEnvVars()
	if err != nil {
		exec.Logger.WithError(err).Error("failed to retrieve credentials for common tf options")
	}

	// set credential environment variables
	for k, v := range creds {
		tfOptions.EnvVars[k] = v
	}

	return
}
