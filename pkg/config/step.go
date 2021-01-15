package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type StepExecution struct {
	RegionDeployType           RegionDeployType
	Region                     string `json:"region"`
	Logger                     *logrus.Entry
	Fs                         afero.Fs
	UniqueExternalExecutionID  string
	RegionGroupRegions         []string
	TerrascaleTargetAccountID  string
	RegionGroup                string
	PrimaryRegion              string
	Dir                        string
	Environment                string `json:"environment"`
	AppVersion                 string `json:"app_version"`
	AccountID                  string `json:"account_id"`
	MaxRetries                 int
	MaxTestRetries             int
	CoreAccounts               map[string]Account
	RegionGroups               RegionGroupsMap
	Namespace                  string
	CommonRegion               string
	StepName                   string
	StepID                     string
	DeploymentRing             string
	Project                    string
	TrackName                  string
	DryRun                     bool
	SelfDestroy                bool
	DefaultStepOutputVariables map[string]map[string]string // Previous step output variables are available in this map. K=StepName,V=map[VarName:VarVal]
	OptionalStepParams         map[string]string
	RequiredStepParams         map[string]interface{}
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
	DeployConfig           Config
	CommonInputVariables   map[string]string // Common input variables that all steps receive
	Output                 StepOutput
	TestOutput             StepTestOutput
	Runner                 Stepper
	//TerrascaleConfig       TerrascaleConfig
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
	PreExecute(execution StepExecution) (exec StepExecution, err error)
	ExecuteStep(execution StepExecution) (resp StepOutput)
	ExecuteStepTests(execution StepExecution) (resp StepTestOutput)
	ExecuteStepDestroy(execution StepExecution) (output StepOutput)
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
