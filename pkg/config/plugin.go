package config

import "github.com/sirupsen/logrus"

// Interface RunnerPlugin describes capacilities and initializtion for runiac plugins.
type RunnerPlugin interface {
	// Initialize allows a plugin to perform one-time initialization prior to use.
	// Any user-facing output should be sent to the provided`logger` instance.
	Initialize(logger *logrus.Entry)
}
