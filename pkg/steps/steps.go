//go:generate mockgen -destination ../../mocks/mock_steps.go -package=mocks github.optum.com/healthcarecloud/terrascale/pkg/steps StepperFactory,Stepper

package steps

import (
	"fmt"
	"strings"

	"github.optum.com/healthcarecloud/terrascale/pkg/config"
)

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

func KeysString(m map[string]config.Account) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}

func KeysStringMap(m map[string]map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}
