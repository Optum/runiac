package steps_test

import (
	"flag"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	plugins_terraform "github.optum.com/healthcarecloud/terrascale/plugins/terraform"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.optum.com/healthcarecloud/terrascale/pkg/steps"
)

var DefaultStubAccountID = "1"
var StubVersion = "v0.0.5"
var logger *logrus.Entry
var sut config.Stepper

func TestMain(m *testing.M) {

	logs := logrus.New()
	logs.SetLevel(logrus.WarnLevel)
	logger = logrus.NewEntry(logs)

	sut = plugins_terraform.TerraformStepper{}

	flag.Parse()
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}

func TestAddTrackOutputToParams(t *testing.T) {
	stepParams := make(map[string]string)
	outputVars := make(map[string]map[string]string)

	outputVars["cool_step1"] = make(map[string]string)
	outputVars["cool_step1"]["k1"] = "v1"
	outputVars["cool_step1"]["k2"] = "v2"
	outputVars["cool_step2"] = make(map[string]string)
	outputVars["cool_step2"]["k3"] = "v3"

	mockParams := steps.AppendToStepParams(stepParams, outputVars)

	keyCount := 0
	for range mockParams {
		keyCount++
	}

	require.Equal(t, 3, keyCount, "There should be a key for every output from every step")
	require.Equal(t, "v1", mockParams["cool_step1-k1"], "stepParams should be set with the correct key and value")
	require.Equal(t, "v2", mockParams["cool_step1-k2"], "stepParams should be set with the correct key and value")
	require.Equal(t, "v3", mockParams["cool_step2-k3"], "stepParams should be set with the correct key and value")
}
