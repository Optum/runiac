package arm

import (
	"github.com/optum/runiac/pkg/shell"
)

func RunAzureCLICommand(streamOutput bool, options *Options, additionalArgs ...string) (string, error) {
	cmd := shell.Command{
		Command:           options.AzureCLIBinary,
		Args:              additionalArgs,
		WorkingDir:        options.AzureCLIDir,
		Env:               options.EnvVars,
		OutputMaxLineSize: options.OutputMaxLineSize,
		NonInteractive:    true,
		SensitiveArgs:     false,
		Logger:            options.Logger,
	}

	if streamOutput {
		return shell.RunShellCommandAndGetAndStreamOutput(cmd)
	}

	return shell.RunShellCommandAndGetOutput(cmd)
}
