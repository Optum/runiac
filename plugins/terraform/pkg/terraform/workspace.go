package terraform

import (
	"fmt"
	"strings"
)

// Plan runs terraform plan with the given options and returns stdout/stderr.
func WorkspaceSelect(options *Options, ws string) (string, error) {
	args := []string{"workspace", "select", ws}

	resp, err := RunTerraformCommand(true, options, FormatArgs(options, args...)...)

	if err != nil && strings.Contains(strings.ToLower(resp), strings.ToLower(fmt.Sprintf("workspace \"%s\" doesn't exist", ws))) {
		argsNew := []string{"workspace", "new", ws}
		resp2, err2 := RunTerraformCommand(true, options, FormatArgs(options, argsNew...)...)

		if err2 != nil {
			return resp2, err2
		}

		return RunTerraformCommand(true, options, FormatArgs(options, args...)...)
	}

	return resp, err
}
