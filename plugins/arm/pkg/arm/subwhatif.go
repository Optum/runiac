package arm

func SubWhatIf(options *Options, deploymentName string, accountID string, location string, file string) (out string, err error) {
	args := []string{
		"deployment",
		"sub",
		"what-if",
		"--name",
		deploymentName,
		"--location",
		location,
		"--template-file",
		file,
		"--subscription",
		accountID,
	}

	return RunAzureCLICommand(true, options, args...)
}
