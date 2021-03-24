package cmd

import "testing"

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
		if result != expected {
			t.Errorf("sanitizeMachineName(\"%s\") = \"%s\"; want \"%s\"", in, result, expected)
		}
	}
}
