package plugins_arm

import (
	"fmt"
	"encoding/json"

	"github.com/spf13/afero"
	"github.com/optum/runiac/pkg/config"
	"github.com/optum/runiac/plugins/arm/pkg/arm"
)

type ArmStepper struct{}

var azureCLI arm.AzureCLI = arm.AzureCLI{}

func (stepper ArmStepper) PreExecute(exec config.StepExecution) (config.StepExecution, error) {
	return exec, nil
}

// ExecuteStepDestroy destroys a step
func (stepper ArmStepper) ExecuteStepDestroy(exec config.StepExecution) (output config.StepOutput) {
	output.RegionDeployType = exec.RegionDeployType
	output.Region = exec.Region
	output.StepName = exec.StepName
	output.Status = config.Fail
	var options *arm.Options
	deploymentName := createDeploymentName(exec)

	options, output.Err = getCommonOptions(exec)
	if output.Err != nil {
		options.Logger.WithError(output.Err).Error("Unable to prepare for step destroy execution")
		return
	}

	// find the metadata associated with the last deployment
	resp, err := azureCLI.SubShow(options, deploymentName, exec.AccountID)
	if err != nil {
		options.Logger.WithError(output.Err).Error("Failed to plan template deployment")
		return
	}

	// unmarshal the output into a struct
	metadata := deployment{}
	err = json.Unmarshal([]byte(resp), &metadata)
	if err != nil {
		options.Logger.WithError(output.Err).Error("Failed to read last template deployment")
		return
	}

	// gather the list of resource ids affected by that deployment
	ids := []string{}
	for _, resource := range metadata.Properties.OutputResources {
		ids = append(ids, resource.ID)
	}

	// delete all created resources
	_, err = azureCLI.ResourceDelete(options, ids)
	if err != nil {
		options.Logger.WithError(output.Err).Error("Failed to delete resources")
		return
	}

	// delete deployment metadata
	_, err = azureCLI.SubDelete(options, deploymentName, exec.AccountID)
	if err != nil {
		options.Logger.WithError(output.Err).Error("Failed to delete deployment metadata")
		return
	}

	output.Status = config.Success
	return
}

// ExecuteStep deploys a step
func (stepper ArmStepper) ExecuteStep(exec config.StepExecution) (output config.StepOutput) {
	output.RegionDeployType = exec.RegionDeployType
	output.Region = exec.Region
	output.StepName = exec.StepName
	output.Status = config.Fail
	var options *arm.Options
	deploymentName := createDeploymentName(exec)

	options, output.Err = getCommonOptions(exec)
	if output.Err != nil {
		options.Logger.WithError(output.Err).Error("Unable to prepare for step execution")
		return
	}

	mainTemplateFile, err := parseMainTemplate(exec)
	if err != nil {
		options.Logger.WithError(output.Err).Error("Unable to parse template for step execution")
		return
	}

	_, err = azureCLI.SubWhatIf(options, deploymentName, exec.AccountID, exec.Region, mainTemplateFile)
	if err != nil {
		options.Logger.WithError(output.Err).Error("Failed to plan template deployment")
		return
	}

	if exec.DryRun {
		options.Logger.Info("---------- Skipping create, this is a dry run ---------- ")
	} else {
		_, err = azureCLI.SubCreate(options, deploymentName, exec.AccountID, exec.Region, mainTemplateFile)
		if err != nil {
			options.Logger.WithError(output.Err).Error("Failed to deploy template")
			return
		}
	}

	output.Status = config.Success
	return
}

// ExecuteStepTests executes the tests for a step
func (stepper ArmStepper) ExecuteStepTests(exec config.StepExecution) (output config.StepTestOutput) {
	// TODO
	return
}

func getCommonOptions(exec config.StepExecution) (options *arm.Options, err error) {
	options = &arm.Options{
		AzureCLIBinary:           "az",
		AzureCLIDir:              exec.Dir,
		EnvVars:                  map[string]string{},
		Logger:                   exec.Logger,
	}

	return
}

func createDeploymentName(exec config.StepExecution) string {
	return fmt.Sprintf("runiac-%s-%s-%s-%s", exec.Project, exec.TrackName, exec.StepName, exec.Region)
}

func parseMainTemplate(exec config.StepExecution) (string, error) {
	fs := afero.NewOsFs()
	
	err := fs.Mkdir(fmt.Sprintf("%s/.temp", exec.Dir), 0755)
	if err != nil {
		exec.Logger.WithError(err).Error(err)
		return "", err
	}

	data, err := afero.ReadFile(fs, fmt.Sprintf("%s/main.json", exec.Dir))
	if err != nil {
		exec.Logger.WithError(err).Error(err)
		return "", err
	}

	template := make(map[string]interface{})
	err = json.Unmarshal(data, &template)
	if err != nil {
		exec.Logger.Error("Unable to unmarshall template file")
		exec.Logger.WithError(err).Error(err)
		return "", err
	}

	resources := template["resources"].([]interface{})
	for _, item := range resources {
		resource := item.(map[string]interface{})

		resourceType := resource["type"].(string)
		if resourceType != "Microsoft.Resources/deployments" {
			continue
		}

		properties := resource["properties"].(map[string]interface{})
		if properties == nil {
			continue
		}

		if properties["_templateLink"] == nil {
			continue
		}

		templateLink := properties["_templateLink"].(map[string]interface{})
		if templateLink != nil {
			localUri := templateLink["localUri"].(string)
			
			linkedTemplateData, err := afero.ReadFile(fs, fmt.Sprintf("%s/%s", exec.Dir, localUri))
			if err != nil {
				exec.Logger.WithError(err).Error(err)
				return "", err
			}

			linkedTemplate := make(map[string]interface{})
			err = json.Unmarshal(linkedTemplateData, &linkedTemplate)
			if err != nil {
				exec.Logger.WithError(err).Error(err)
				return "", err
			}

			// replace the _templateLink property with the contents of the template itself
			delete(properties, "_templateLink")
			properties["template"] = linkedTemplate
		}
	}

	result, err := json.Marshal(template)
	if err != nil {
		exec.Logger.Error("Failed to marshal interpolated template")
		exec.Logger.WithError(err).Error(err)
		return "", err
	}

	filePath := ".temp/main.json"
	path := fmt.Sprintf("%s/%s", exec.Dir, filePath)
	err = afero.WriteFile(fs, path, result, 0644)
	if err != nil {
		exec.Logger.WithError(err).Error(err)
		return "", err
	}

	return filePath, nil
}
