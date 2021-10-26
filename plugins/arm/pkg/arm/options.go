package arm

import (
	"github.com/sirupsen/logrus"
)

type Options struct {
	AzureCLIBinary    string
	AzureCLIDir       string
	EnvVars           map[string]string
	OutputMaxLineSize int
	Logger            *logrus.Entry
}
