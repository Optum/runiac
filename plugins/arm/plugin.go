package plugins_arm

import (
	"github.com/sirupsen/logrus"
	"github.com/optum/runiac/plugins/arm/pkg/arm"
)

type ArmPlugin struct {}

func (info ArmPlugin) Initialize(logger *logrus.Entry) {
	logger.Info("Initializing runiac ARM plugin")
	logger.Warn("The ARM runner is currently in preview and is subject to change in future runiac releases")

	// display azure cli binary information
	azureCLI := arm.AzureCLI{}
	
	options := &arm.Options{
		AzureCLIBinary:           "az",
		AzureCLIDir:              ".",
		EnvVars:                  map[string]string{},
		Logger:                   logger.WithField("ArmPlugin", "info"),
	}

	azureCLI.Version(options)
}
