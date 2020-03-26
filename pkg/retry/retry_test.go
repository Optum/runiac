package retry_test

import (
	"errors"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.optum.com/healthcarecloud/terrascale/pkg/retry"
	"github.optum.com/healthcarecloud/terrascale/pkg/steps"
	"os"
	"testing"
	"time"
)

var DefaultStubAccountID = "1"
var StubVersion = "v0.0.5"
var logger *logrus.Entry
var sut steps.Stepper
var stubBackendParserResponse steps.TerraformBackend

func TestMain(m *testing.M) {

	log := logrus.New()
	log.Level = logrus.InfoLevel
	logger = log.WithField("", "")

	stubBackendParserResponse = steps.TerraformBackend{}
	sut = steps.TerraformStepper{}

	flag.Parse()
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}

func TestDoRetry_ShouldParseS3Correctly(t *testing.T) {
	t.Parallel()
	var i int

	_ = retry.DoWithRetry("terraform plan and apply", 3, 1*time.Millisecond, logger, func(attempt int) error {

		require.Equal(t, i, attempt, "attempt should increment")
		i++
		return errors.New("error")
	})
}
