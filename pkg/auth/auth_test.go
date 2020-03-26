package auth_test

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.optum.com/healthcarecloud/terrascale/pkg/auth"
	"os"
	"testing"
)

var logger *logrus.Entry

func TestMain(m *testing.M) {
	logs := logrus.New()
	logs.SetReportCaller(true)
	logger = logs.WithField("environment", "unittest")

	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestGetCredentialEnvVarsForAccount_ReturnsAZUCredsFromCacheIfTheyExist(t *testing.T) {
	credsMap := make(map[string]*auth.AZUCredentials)
	credsMap["123-456"] = &auth.AZUCredentials{
		ID:     "mock-id",
		Secret: "mock-secret",
		Tenant: "mock-tenant",
	}

	auth := &auth.SDKAuthenticator{
		Logger:              logger,
		BedrockCommonRegion: "us-east-1",
		AzuCredCache:        credsMap,
	}

	creds, err := auth.GetCredentialEnvVarsForAccount(auth.Logger, "AZU", "123-456", "UHG")

	assert.Nil(t, err, "There should be no error")
	assert.NotNil(t, creds, "There should be credentials returned")
	assert.Equal(t, creds["ARM_CLIENT_ID"], "mock-id", "The client id should match")
	assert.Equal(t, creds["ARM_CLIENT_SECRET"], "mock-secret", "The client secret should match")
	assert.Equal(t, creds["ARM_TENANT_ID"], "mock-tenant", "The tenant id should match")
	assert.Equal(t, creds["ARM_SUBSCRIPTION_ID"], "123-456", "The subscription id should match")
}
