package terraform

// Apply runs terraform apply with the given options and return stdout/stderr. Note that this method does NOT call destroy and
// assumes the caller is responsible for cleaning up any resources created by running apply.
func Apply(options *Options, tfplan string) (string, error) {
	args := []string{"apply", "-input=false", "-no-color", "-auto-approve=true", tfplan}
	return RunTerraformCommand(true, options, FormatArgs(options, args...)...)
}
