package terraform

import "fmt"

// Plan runs terraform plan with the given options and returns stdout/stderr.
func Plan(options *Options, tfplan string, destroy bool) (string, error) {
	args := []string{"plan", fmt.Sprintf("-out=%s", tfplan), "-input=false", "-no-color"}

	if destroy {
		args = append(args, "-destroy")
	}

	return RunTerraformCommand(true, options, FormatArgs(options, args...)...)
}
