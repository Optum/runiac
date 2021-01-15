package retry_test

import (
	"errors"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.optum.com/healthcarecloud/terrascale/pkg/retry"
	"os"
	"testing"
	"time"
)

var logger *logrus.Entry

func TestMain(m *testing.M) {

	log := logrus.New()
	log.Level = logrus.InfoLevel
	logger = log.WithField("", "")

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
