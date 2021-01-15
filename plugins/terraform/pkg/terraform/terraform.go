// Package terraform allows to interact with Terraform.
package terraform

// This code follows: https://github.com/gruntwork-io/terratest/blob/master/modules/terraform/terraform.go

// https://www.terraform.io/docs/commands/plan.html#detailed-exitcode

// TerraformPlanChangesPresentExitCode is the exit code returned by terraform plan detailed exitcode when changes are present
const TerraformPlanChangesPresentExitCode = 2

// DefaultSuccessExitCode is the exit code returned when terraform command succeeds
const DefaultSuccessExitCode = 0

// DefaultErrorExitCode is the exit code returned when terraform command fails
const DefaultErrorExitCode = 1

type Terraformer interface {
	Version(options *Options) (out string, err error)
	Show(options *Options, tfplan string) (string, error)
	Plan(options *Options, tfplan string, destroy bool) (string, error)
	OutputAll(options *Options) (map[string]interface{}, error)
	OutputForKeysE(options *Options, keys []string) (map[string]interface{}, error)
	OutputToString(value interface{}) string
	Init(options *Options) (out string, err error)
	Apply(options *Options, tfplan string) (string, error)
	WorkspaceSelect(options *Options, workspace string) (string, error)
}

type Terraform struct{}

func (t Terraform) Version(options *Options) (out string, err error) {
	return Version(options)
}

func (t Terraform) Show(options *Options, tfplan string) (string, error) {
	return Show(options, tfplan)
}

func (t Terraform) Plan(options *Options, tfplan string, destroy bool) (string, error) {
	return Plan(options, tfplan, destroy)
}

func (t Terraform) OutputAll(options *Options) (map[string]interface{}, error) {
	return OutputAll(options)
}

func (t Terraform) OutputForKeysE(options *Options, keys []string) (map[string]interface{}, error) {
	return OutputForKeysE(options, keys)
}

func (t Terraform) Init(options *Options) (out string, err error) {
	return Init(options)
}

func (t Terraform) Apply(options *Options, tfplan string) (string, error) {
	return Apply(options, tfplan)
}

func (t Terraform) OutputToString(value interface{}) string {
	return OutputToString(value)
}

func (t Terraform) WorkspaceSelect(options *Options, workspace string) (string, error) {
	return WorkspaceSelect(options, workspace)
}
