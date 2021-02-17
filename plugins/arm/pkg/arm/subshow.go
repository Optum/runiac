package arm

func SubShow(options *Options, deploymentName string, accountID string) (out string, err error) {
	args := []string {
		"deployment",
		"sub",
		"show",
		"--name",
		deploymentName,
		"--subscription",
		accountID,
	}

	return RunAzureCLICommand(true, options, args...)
}
