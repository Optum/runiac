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

func TestGetDockerfileForBuild(t *testing.T) {
	result := getDockerfileForBuild()
	require.Equal(t, ".runiac/Dockerfile", result)
	viper.Set("dockerfile", "mock")
	result2 := getDockerfileForBuild()
	require.Equal(t, "mock", result2)
	viper.Set("dockerfile", "")
}
