package terraform

// This code follows: https://github.com/gruntwork-io/terratest/blob/master/modules/terraform/options.go

import (
	"github.com/sirupsen/logrus"
	"time"
)

// Options for running Terraform commands
type Options struct {
	TerraformBinary          string                 // Name of the binary that will be used
	TerraformDir             string                 // The path to the folder where the Terraform code is defined.
	Vars                     map[string]interface{} // The vars to pass to Terraform commands using the -var option.
	VarFiles                 []string               // The var file paths to pass to Terraform commands using -var-file option.
	Targets                  []string               // The target resources to pass to the terraform command with -target
	EnvVars                  map[string]string      // Environment variables to set when running Terraform
	BackendConfig            map[string]interface{} // The vars to pass to the terraform init command for extra configuration for the backend
	RetryableTerraformErrors map[string]string      // If Terraform apply fails with one of these (transient) errors, retry. The keys are a regexp to match against the error and the message is what to display to a user if that error is matched.
	MaxRetries               int                    // Maximum number of times to retry errors matching RetryableTerraformErrors
	TimeBetweenRetries       time.Duration          // The amount of time to wait between retries
	Upgrade                  bool                   // Whether the -upgrade flag of the terraform init command should be set to true or not
	NoColor                  bool                   // Whether the -no-color flag will be set for any Terraform command or not
	NoStderr                 bool                   // Disable stderr redirection
	OutputMaxLineSize        int                    // The max size of one line in stdout and stderr (in bytes)
	Logger                   *logrus.Entry
	PluginCacheDir           string
}
