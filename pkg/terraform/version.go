package terraform

// displays terraform version string
func Version(options *Options) (string, error) {
	args := []string{"version"}

	return RunTerraformCommand(false, options, FormatArgs(options, args...)...)
}
