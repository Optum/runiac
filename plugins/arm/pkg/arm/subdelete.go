package arm

func SubDelete(options *Options, deploymentName string, accountID string) (out string, err error) {
	args := []string{
		"deployment",
		"sub",
		"delete",
		"--name",
		deploymentName,
		"--subscription",
		accountID,
	}

	return RunAzureCLICommand(true, options, args...)
}
