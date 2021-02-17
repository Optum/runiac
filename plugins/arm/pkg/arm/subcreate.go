package arm

func SubCreate(options *Options, deploymentName string, accountID string, location string, file string) (out string, err error) {
	args := []string{
		"deployment",
		"sub",
		"create",
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
