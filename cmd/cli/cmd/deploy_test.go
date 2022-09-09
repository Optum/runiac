package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestSanitizeMachinename(t *testing.T) {
	tests := map[string]string{
		"foobar":            "foobar",
		"foo()bar":          "foo__bar",
		"!234 Test":         "_234_Test",
		"!@#$%^&*()":        "__________",
		"trailing space ":   "trailing_space",
		"\nwhite space\n\t": "white_space",
		"domain\\user":      "domain_user",
	}

	for in, expected := range tests {
		result := sanitizeMachineName(in)

		require.Equal(t, expected, result, "sanitizeMachineName(\"%s\") = \"%s\"; want \"%s\"", in, result, expected)
	}
}

func TestGetBuildArguments_ShouldSetBuildArgContainerOnlyWhenValueExists(t *testing.T) {
	// if container is set, include in docker build. if not, do not include.
	tests := map[string][]string{
		"foobar": {"--build-arg", "RUNIAC_CONTAINER=foobar", "."},
		"":       {"."},
	}

	for in, expected := range tests {
		Container = in

		result := getBuildArguments()
		require.Equal(t, expected, result)
	}
}

func Test_DeployCommand(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"deploy", "--test"})
	cmd.Execute()

	// Assert config values are used when command line is not present
	cmd.SetArgs([]string{"deploy", "--test"})
	viper.Set("container_engine", "mock")
	viper.Set("container", "mock")
	viper.Set("dockerfile", "mock")

	cmd.Execute()
	require.Equal(t, "mock", Dockerfile)
	require.Equal(t, "mock", ContainerEngine)
	require.Equal(t, "mock", Container)

	// Assert command line precedence
	cmd.SetArgs([]string{"deploy", "--test", "--dockerfile=mockofseagulls", "--container-engine=mockofseagulls", "--container=mockofseagulls"})
	viper.Set("container_engine", "ignore_me")
	viper.Set("container", "ignore_me")
	viper.Set("dockerfile", "ignore_me")

	cmd.Execute()
	require.Equal(t, "mockofseagulls", Dockerfile)
	require.Equal(t, "mockofseagulls", ContainerEngine)
	require.Equal(t, "mockofseagulls", Container)
}
