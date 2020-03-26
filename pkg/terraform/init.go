package terraform

import (
	"strings"
	"time"

	"github.optum.com/healthcarecloud/terrascale/pkg/retry"
)

// Init calls terraform init and return stdout/stderr.
func Init(options *Options) (out string, err error) {
	args := []string{"init", "-force-copy", "-get-plugins=false"}
	backendArgs := FormatTerraformBackendConfigAsArgs(options.BackendConfig)
	args = append(args, backendArgs...)

	options.Logger.Infof("BackendConfig: %v", strings.Join(backendArgs, " "))
	retryErr := retry.DoWithRetry("terraform init", 3, 10*time.Second, options.Logger, func(attempt int) error {
		out, err = RunTerraformCommand(true, options, args...)

		return err
	})

	if retryErr != nil {
		options.Logger.WithError(retryErr).Error("Error attempting retryable action")
	}

	return

}
