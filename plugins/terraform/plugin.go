package plugins_terraform

import (
	"github.com/optum/runiac/plugins/terraform/pkg/terraform"
	"github.com/sirupsen/logrus"
)

type TerraformPlugin struct {}

func (info TerraformPlugin) Initialize(logger *logrus.Entry) {
	logger.Info("Initializing runiac Terraform plugin")

	// display terraform binary information
	// disable checkpoints since we just want to print the version string alone
	tfOptions := &terraform.Options{
		TerraformDir: ".",
		EnvVars: map[string]string{
			"CHECKPOINT_DISABLE": "true",
		},
		Logger:             logger.WithField("terraform", "version"),
		NoColor:            true,
		MaxRetries:         1,
		TimeBetweenRetries: 0,
	}

	terraformer := &terraform.Terraform{}
	resp, err := terraformer.Version(tfOptions)
	if err != nil {
		tfOptions.Logger.WithError(err).Error("Error running terraform version")
	} else {
		tfOptions.Logger.Info("Binary: ", resp)
	}
}
