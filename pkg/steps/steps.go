//go:generate mockgen -destination ../../mocks/mock_steps.go -package=mocks github.optum.com/healthcarecloud/terrascale/pkg/steps StepperFactory,Stepper

package steps

import (
	"fmt"
	"strings"

	"github.optum.com/healthcarecloud/terrascale/pkg/config"
)

// StepperFactory is a modular abstraction for determining step implementation
// also used for mocking step execution in unit tests
type StepperFactory interface {
	Get(s Step) (stepper Stepper)
}

// TerraformOnlyStepperFactory implements the StepperFactory interface
type TerraformOnlyStepperFactory struct{}

// Get implements the StepperFactory interface
func (f TerraformOnlyStepperFactory) Get(s Step) (stepper Stepper) {
	return TerraformStepper{}
}

type TerrascaleConfig struct {
	Enabled     bool                        `mapstructure:"enabled"`
	ExecuteWhen TerrascaleConfigExecuteWhen `mapstructure:"execute_when"`
}

type TerrascaleConfigExecuteWhen struct {
	RegionIn []string `mapstructure:"region_in"`
}

// Step represents a delivery framework step, e.g. the executions needed to implement a track
type Step struct {
	ID                     string
	Name                   string
	TrackName              string
	Dir                    string
	ProgressionLevel       int // 1, 2, 3...
	RegionalResourcesExist bool
	TestsExist             bool
	RegionalTestsExist     bool // TODO: remove the need for these TestsExists and evaulate in real time during evaluation vs gather?
	DeployConfig           config.Config
	CommonInputVariables   map[string]string // Common input variables that all steps receive
	Output                 StepOutput
	TestOutput             StepTestOutput
	TFProvider             TerraformProvider
	TFBackend              TerraformBackend
	TerrascaleConfig       TerrascaleConfig
}

// StepTestOutput represents the output of a step's test
type StepTestOutput struct {
	StepName     string
	StreamOutput string
	Err          error
}

// StepOutput represents the output of a step
type StepOutput struct {
	Status           DeployResult
	RegionDeployType RegionDeployType
	Region           string
	StepName         string
	StreamOutput     string
	Err              error
	OutputVariables  map[string]interface{}
}

// TFProviderType represents a Terraform provider type
type RegionDeployType int

const (
	// Primary region typedeploys to the designated primary region, this usually consists of global resources such as IAM
	// In Terrascale world, this means it would only deploy the step's parent directory resources
	PrimaryRegionDeployType RegionDeployType = iota
	// Regional region type deploys to each of the targeted regions, this consists of region specific resources and does not include global resources such as IAM
	// In Terrascale world, this means it would only deploy the step's /regional/ directory resources to each of the targeted regions
	RegionalRegionDeployType
)

func (p RegionDeployType) String() string {
	return [...]string{"primary", "regional"}[p]
}

// Stepper is an interface for working with delivery framework steps, e.g. the executions needed to implement a track
// All Step methods will handle logging of errors while logger has appropriate fields set.
// Therefore, there should be no need to logger Output.Errs from this interface
type Stepper interface {
	// ExecuteStep will handle the deployment of this step.  In Terraform this will include init, plan, verify plan, and apply.
	ExecuteStep(execution ExecutionConfig) (resp StepOutput)
	ExecuteStepTests(execution ExecutionConfig) (resp StepTestOutput)
	ExecuteStepDestroy(execution ExecutionConfig) (output StepOutput)
}

type DeployResult int

const (
	Fail DeployResult = iota
	Success
	Unstable
	Skipped
	Na // not applicable (e.g. no regional resources exist or step was disabled for execution)
)

func (d DeployResult) String() string {
	return [...]string{"FAIL", "SUCCESS", "UNSTABLE", "SKIPPED"}[d]
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
