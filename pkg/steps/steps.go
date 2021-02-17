//go:generate mockgen -destination ../../mocks/mock_steps.go -package=mocks github.com/optum/runiac/pkg/config Stepper

package steps

import (
	"fmt"
	pluginsarm "github.com/optum/runiac/plugins/arm"
	pluginsterraform "github.com/optum/runiac/plugins/terraform"
	"strings"

	"github.com/optum/runiac/pkg/config"
)

func DetermineRunner(s config.Step) config.Stepper {
	switch s.DeployConfig.Runner {
	case "arm":
		return pluginsarm.ArmStepper{}
	case "terraform":
		return pluginsterraform.TerraformStepper{}
	default:
		return nil
	}
}

// Adds previous step output to stepParams which get added as environment variables
// during terraform plan
func AppendToStepParams(stepParams map[string]string, incomingOutputVars map[string]map[string]string) map[string]string {
	for stepName, stepOutputMap := range incomingOutputVars {
		for stepOutputVarKey, stepOutputVarValue := range stepOutputMap {
			key := fmt.Sprintf("%s-%s", stepName, stepOutputVarKey)
			stepParams[key] = stepOutputVarValue
		}
	}
	return stepParams
}

func KeysStringMap(m map[string]map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}
