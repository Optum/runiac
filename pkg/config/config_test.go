package config_test

import (
	"flag"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var DefaultStubAccountID = "1"
var StubVersion = "v0.0.5"

var validate *validator.Validate
var fs afero.Fs
var log *logrus.Entry

func TestMain(m *testing.M) {
	os.Setenv("ACCOUNT_ID", DefaultStubAccountID)
	os.Setenv("CSP", "AWS")
	os.Setenv("CREDS_ID", "1")
	os.Setenv("CODEPIPELINE_EXECUTION_ID", "1")
	os.Setenv("STEP_FUNCTION_NAME", "1")
	os.Setenv("UPDATE_STATUS_LAMBDA", "1")

	os.Setenv("ENVIRONMENT", "local")
	os.Setenv("NAMESPACE", "1")
	os.Setenv("DEPLOYMENT_RING", "INTERNAL")

	validate = validator.New()
	fs = afero.NewMemMapFs()
	log = logrus.New().WithField("environment", "unittest")

	var stubVersionJSON = []byte(fmt.Sprintf(`{
	  "version": "%s",
	  "pr_region": "",
	  "base_image": "common-bedrock-customer-deploy:v0.0.3"
	}`, StubVersion))

	afero.WriteFile(fs, "version.json", stubVersionJSON, 0644)

	flag.Parse()
	exitCode := m.Run()

	_ = os.Remove("testbackend.tf")

	// Exit
	os.Exit(exitCode)
}

func TestGetConfig_ShouldParseEnvironmentVariables(t *testing.T) {
	mock, err := config.GetConfig()
	assert.Equal(t, DefaultStubAccountID, mock.AccountID, "Account ID should be set correctly")
	assert.Nil(t, err, "err should return nil")
}

func TestGetConfig_ShouldAllowLocalDeploymentRing(t *testing.T) {
	oldRing := os.Getenv("DEPLOYMENT_RING")
	os.Setenv("DEPLOYMENT_RING", "LOCAL")
	_, err := config.GetConfig()
	assert.Nil(t, err)
	os.Setenv("DEPLOYMENT_RING", oldRing)
}

func TestGetConfig_ShouldAllowProdDeploymentRing(t *testing.T) {
	oldRing := os.Getenv("DEPLOYMENT_RING")
	os.Setenv("DEPLOYMENT_RING", "PROD")
	_, err := config.GetConfig()
	assert.Nil(t, err)
	os.Setenv("DEPLOYMENT_RING", oldRing)
}

func TestGetVersionJSON_ShouldParseVersionFile(t *testing.T) {
	mock, err := config.GetVersionJSON(log, fs, "version.json")
	assert.Equal(t, StubVersion, mock.Version, "Version should be set correctly")
	assert.Nil(t, err, "err should return nil")
}

func TestGetConfig_DefaultRegionGroupsExist(t *testing.T) {

	config, err := config.GetConfig()

	require.NoError(t, err)
	require.NotEmpty(t, config.RegionGroups)
}
