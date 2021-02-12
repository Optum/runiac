package arm

func Version(options *Options) (out string, err error) {
	args := []string{
		"--version",
	}

	return RunAzureCLICommand(true, options, args...)
}
