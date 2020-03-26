package terraform

// This code follows: https://github.com/gruntwork-io/terratest/blob/master/modules/terraform/cmd.go

import (
	"strings"

	"github.com/gruntwork-io/terratest/modules/collections"
	"github.optum.com/healthcarecloud/terrascale/pkg/shell"
)

// GetCommonOptions extracts commons terraform options
func GetCommonOptions(options *Options, args ...string) (*Options, []string) {
	if options.NoColor && !collections.ListContains(args, "-no-color") {
		args = append(args, "-no-color")
	}

	if options.PluginCacheDir != "" {
		if options.EnvVars == nil {
			options.EnvVars = map[string]string{}
		}
		options.EnvVars["TF_PLUGIN_CACHE_DIR"] = options.PluginCacheDir
	}

	if options.TerraformBinary == "" {
		options.TerraformBinary = "terraform"
	}

	return options, args
}

// RunTerraformCommand runs terraform with the given arguments and options and return stdout/stderr.
func RunTerraformCommand(streamOutput bool, additionalOptions *Options, additionalArgs ...string) (string, error) {
	options, args := GetCommonOptions(additionalOptions, additionalArgs...)

	cmd := shell.Command{
		Command:           options.TerraformBinary,
		Args:              args,
		WorkingDir:        options.TerraformDir,
		Env:               options.EnvVars,
		OutputMaxLineSize: options.OutputMaxLineSize,
		NonInteractive:    true,
		SensitiveArgs:     false,
		Logger:            options.Logger,
	}

	options.Logger.Infof("Executing Command with following Env Vars set: %s", KeysStringString(cmd.Env))

	if streamOutput {
		return shell.RunShellCommandAndGetAndStreamOutput(cmd)
	}
	return shell.RunShellCommandAndGetOutput(cmd)
}

// KeysStringString returns a string representation of all keys in the map
func KeysStringString(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}

// GetExitCodeForTerraformCommand runs terraform with the given arguments and options and returns exit code
func GetExitCodeForTerraformCommand(additionalOptions *Options, additionalArgs ...string) (int, error) {
	options, args := GetCommonOptions(additionalOptions, additionalArgs...)

	cmd := shell.Command{
		Command:           options.TerraformBinary,
		Args:              args,
		WorkingDir:        options.TerraformDir,
		Env:               options.EnvVars,
		OutputMaxLineSize: options.OutputMaxLineSize,
		Logger:            options.Logger,
		NonInteractive:    true,
		SensitiveArgs:     true,
	}

	_, err := shell.RunShellCommandAndGetOutput(cmd)
	if err == nil {
		return DefaultSuccessExitCode, nil
	}
	exitCode, getExitCodeErr := shell.GetExitCodeForRunCommandError(err)
	if getExitCodeErr == nil {
		return exitCode, nil
	}
	return DefaultErrorExitCode, getExitCodeErr
}
