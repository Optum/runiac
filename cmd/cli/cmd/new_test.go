package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
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
