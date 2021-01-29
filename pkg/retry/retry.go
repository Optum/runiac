package retry

// This code follows: https://github.com/gruntwork-io/terratest/blob/master/modules/retry/retry.go

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

// DoWithRetry runs the specified action. If it returns a value, return that value. If it returns an error, sleep for
// sleepBetweenRetries and try again, up to a maximum of maxRetries retries. If maxRetries is exceeded, return a
// MaxRetriesExceeded error.
func DoWithRetry(actionDescription string, maxRetries int, sleepBetweenRetries time.Duration, logger *logrus.Entry, action func(attempt int) error) error {
	for i := 0; i <= maxRetries; i++ {
		logger.Infof(actionDescription)

		err := action(i)
		if err == nil {
			return nil
		}

		// don't sleep after the final retry attempt
		if i < maxRetries {
			logger.WithError(err).Warningf("%s returned an error: %s. Sleeping for %s and will try again. Retry Count: %v.", actionDescription, err.Error(), sleepBetweenRetries, i)
			time.Sleep(sleepBetweenRetries)
		} else {
			logger.WithError(err).Warningf("%s returned an error: %s. Retry Count: %v.", actionDescription, err.Error(), i)
		}
	}

	return MaxRetriesExceeded{Description: actionDescription, MaxRetries: maxRetries}
}

// MaxRetriesExceeded is an error that occurs when the maximum amount of retries is exceeded.
type MaxRetriesExceeded struct {
	Description string
	MaxRetries  int
}

func (err MaxRetriesExceeded) Error() string {
	return fmt.Sprintf("'%s' unsuccessful after %d retries", err.Description, err.MaxRetries)
}
