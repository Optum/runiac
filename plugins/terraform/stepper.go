package plugins_terraform

import (
	"encoding/json"
	"fmt"
	"github.com/optum/runiac/pkg/config"
	"github.com/optum/runiac/pkg/retry"
	"github.com/optum/runiac/pkg/shell"
	"github.com/optum/runiac/plugins/terraform/pkg/terraform"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type TerraformStepper struct{}

var terraformer terraform.Terraformer = terraform.Terraform{}

func (stepper TerraformStepper) PreExecute(exec config.StepExecution) (config.StepExecution, error) {
	HandleDeployOverrides(exec.Logger, exec.Dir, exec.DeploymentRing)

	if exec.SelfDestroy {
		HandleDestroyOverrides(exec.Logger, exec.Dir, exec.DeploymentRing)
	}

	return exec, nil
}

// ExecuteStepDestroy destroys a step
func (stepper TerraformStepper) ExecuteStepDestroy(exec config.StepExecution) config.StepOutput {
	return executeTerraformInDir(exec, true)
}

// ExecuteStep deploys a step
func (stepper TerraformStepper) ExecuteStep(exec config.StepExecution) config.StepOutput {
	return executeTerraformInDir(exec, false)
}

// ExecuteStepTests executes the tests for a step
func (stepper TerraformStepper) ExecuteStepTests(exec config.StepExecution) (output config.StepTestOutput) {
	HandleDeployOverrides(exec.Logger, exec.Dir, exec.DeploymentRing)

	envVars := map[string]string{}

	for k, v := range GetTerraformEnvVars(exec) {
		envVars[fmt.Sprintf("TF_VAR_%s", k)] = v
	}

	for k, v := range GetTerraformCLIVars(exec) {
		envVars[fmt.Sprintf("TF_VAR_%s", k)] = fmt.Sprintf("%v", v)
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

	return
}

func GetTerraformCLIVars(exec config.StepExecution) map[string]interface{} {
	vars := map[string]interface{}{
		"runiac_environment": exec.Environment,
		"runiac_account_id":  exec.AccountID,
		"runiac_region":      exec.Region,
	}

	return vars
}

func GetTerraformEnvVars(exec config.StepExecution) map[string]string {
	output := exec.OptionalStepParams
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

		output["runiac_core_account_ids_map"] = coreAccounts
	}

	output["runiac_app_version"] = exec.AppVersion
	output["runiac_namespace"] = exec.Namespace

	return output
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
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

// executeTerraformInDir is a helper function for executing terraform in a specified directory
var executeTerraformInDir = func(exec config.StepExecution, destroy bool) (output config.StepOutput) {
	output.RegionDeployType = exec.RegionDeployType
	output.Region = exec.Region
	output.StepName = exec.StepName
	output.Status = config.Fail // assume failure
	var resp string
	var tfOptions *terraform.Options

	// terraform init
	tfOptions, output.Err = getCommonTfOptions2(exec)

	if output.Err != nil {
		tfOptions.Logger.WithError(output.Err).Error("unable to retrieve credentials for terraform init")
		return
	}

	tfOptions.BackendConfig = GetBackendConfig(exec, ParseTFBackend).Config
	tfOptions.Logger = tfOptions.Logger.WithField("terraform", "init")
	resp, output.Err = terraformer.Init(tfOptions)

	if output.Err != nil {
		tfOptions.Logger.WithError(output.Err).Error("Error during terraform init")
		return
	}

	tfOptions.Logger = tfOptions.Logger.WithField("terraform", "workspace")

	workspace := fmt.Sprintf("%s-%s", exec.RegionDeployType.String(), exec.Region)

	if exec.Namespace != "" {
		workspace = fmt.Sprintf("%s-%s", exec.Namespace, workspace)
	}
	resp, output.Err = terraformer.WorkspaceSelect(tfOptions, workspace)

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
		for k, v := range GetTerraformEnvVars(exec) {
			tfOptions.Logger.Debugf("Adding parameter to TF_VARs: %s", k)
			tfOptions.EnvVars[fmt.Sprintf("TF_VAR_%s", k)] = v
		}

		tfOptions.Vars = GetTerraformCLIVars(exec)

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

		output.Status = config.Success

		return nil
	})

	return
}

// GetBackendConfig parses a backend.tf file
// TODO, replace this with a cleaner hcl2json2struct merge where backend.tf configurations take priority over defined defaults here
func GetBackendConfig(exec config.StepExecution, backendParser TFBackendParser) TerraformBackend {
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

func interpolateString(exec config.StepExecution, s string) string {
	if strings.Contains(s, "${var.runiac_deployment_ring}") {
		s = strings.ReplaceAll(s, "${var.runiac_deployment_ring}", exec.DeploymentRing)
	}

	if strings.Contains(s, "${var.runiac_target_account_id}") {
		s = strings.ReplaceAll(s,
			"${var.runiac_target_account_id}", exec.TargetAccountID)
	}

	if strings.Contains(s, "${var.runiac_step}") {
		s = strings.ReplaceAll(s,
			"${var.runiac_step}", exec.StepName)
	}

	if strings.Contains(s, "${var.runiac_region_deploy_type}") {
		s = strings.ReplaceAll(s,
			"${var.runiac_region_deploy_type}", exec.RegionDeployType.String())
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
			"${var.runiac_step}", exec.StepName)
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

func getCommonTfOptions2(exec config.StepExecution) (tfOptions *terraform.Options, err error) {
	tfOptions = &terraform.Options{
		TerraformDir:             exec.Dir,
		EnvVars:                  map[string]string{},
		Logger:                   exec.Logger,
		NoColor:                  true,
		RetryableTerraformErrors: map[string]string{".*": "General Terraform error occurred."},
		MaxRetries:               exec.MaxRetries,
		TimeBetweenRetries:       5 * time.Second,
	}

	return
}
