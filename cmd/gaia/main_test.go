package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.optum.com/healthcarecloud/terrascale/pkg/config"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

var DefaultStubAccountID = "1"
var StubVersion = "v0.0.5"

func TestMain(m *testing.M) {
	os.Setenv("ACCOUNT_ID", DefaultStubAccountID)
	os.Setenv("CSP", "AWS")
	os.Setenv("CREDS_ID", "1")
	os.Setenv("CODEPIPELINE_EXECUTION_ID", "1")
	os.Setenv("STEP_FUNCTION_NAME", "1")
	os.Setenv("UPDATE_STATUS_LAMBDA", "1")

	os.Setenv("ENVIRONMENT", "local")
	os.Setenv("NAMESPACE", "1")

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

func TestOverridePRDeployment_OverridesValuesForPullRequests(t *testing.T) {
	tests := []struct {
		Deployment            config.Deployment
		ExpectedPrimaryRegion string
		ExpectedTargetRegions []string
	}{
		{
			Deployment: config.Deployment{
				DeployMetadata: config.DeployMetadata{
					Version:   "pr-10",
					Region:    "centralus,eastus2",
					BaseImage: "base_image",
				},
				Config: config.Config{},
			},
			ExpectedPrimaryRegion: "centralus",
			ExpectedTargetRegions: []string{"centralus", "eastus2"},
		},
		{
			Deployment: config.Deployment{
				DeployMetadata: config.DeployMetadata{
					Version:   "pr-27",
					Region:    "us-east-2",
					BaseImage: "base_image",
				},
				Config: config.Config{},
			},
			ExpectedPrimaryRegion: "us-east-2",
			ExpectedTargetRegions: []string{"us-east-2"},
		},
	}

	for _, tc := range tests {
		// Execute
		overridePRDeployment(&tc.Deployment)

		// Assert
		require.Equal(t, tc.ExpectedPrimaryRegion, tc.Deployment.Config.GaiaPrimaryRegionOverride, "The primary region should be overriden with the correct value")
		require.Equal(t, tc.ExpectedTargetRegions, tc.Deployment.Config.GaiaTargetRegions, "The target regions should be overriden with the correct value")
	}
}
