package terraform

// Show runs terraform show and returns the output and any error
func Show(options *Options, tfplan string) (string, error) {
	args := []string{"show", "-json", tfplan}

	return RunTerraformCommand(false, options, FormatArgs(options, args...)...)
}
