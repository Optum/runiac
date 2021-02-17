package arm

func ResourceDelete(options *Options, ids []string) (out string, err error) {
	args := []string {
		"resource",
		"delete",
		"--ids",
	}

	args = append(args, ids...)

	return RunAzureCLICommand(true, options, args...)
}
