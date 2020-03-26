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
	"gopkg.in/h2non/gock.v1"
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

func TestGetRunningFargateTaskID_ShouldReturnCorrectTaskID(t *testing.T) {
	defer gock.Off()

	stubTaskId := "9781c248-0edd-4cdb-9a93-f63cb662a5d3"

	gock.New(config.FargateTaskMetadataEndpoint).
		Reply(200).
		BodyString(fmt.Sprintf(`{
		  "Cluster": "default",
		  "TaskARN": "arn:aws:ecs:us-east-2:012345678910:task/%s",
		  "Family": "nginx",
		  "Revision": "5",
		  "DesiredStatus": "RUNNING",
		  "KnownStatus": "RUNNING",
		  "Containers": [
			{
			  "DockerId": "731a0d6a3b4210e2448339bc7015aaa79bfe4fa256384f4102db86ef94cbbc4c",
			  "Name": "~internal~ecs~pause",
			  "DockerName": "ecs-nginx-5-internalecspause-acc699c0cbf2d6d11700",
			  "Image": "amazon/amazon-ecs-pause:0.1.0",
			  "ImageID": "",
			  "Labels": {
				"com.amazonaws.ecs.cluster": "default",
				"com.amazonaws.ecs.container-name": "~internal~ecs~pause",
				"com.amazonaws.ecs.task-arn": "arn:aws:ecs:us-east-2:012345678910:task/9781c248-0edd-4cdb-9a93-f63cb662a5d3",
				"com.amazonaws.ecs.task-definition-family": "nginx",
				"com.amazonaws.ecs.task-definition-version": "5"
			  },
			  "DesiredStatus": "RESOURCES_PROVISIONED",
			  "KnownStatus": "RESOURCES_PROVISIONED",
			  "Limits": {
				"CPU": 0,
				"Memory": 0
			  },
			  "CreatedAt": "2018-02-01T20:55:08.366329616Z",
			  "StartedAt": "2018-02-01T20:55:09.058354915Z",
			  "Type": "CNI_PAUSE",
			  "Networks": [
				{
				  "NetworkMode": "awsvpc",
				  "IPv4Addresses": [
					"10.0.2.106"
				  ]
				}
			  ]
			},
			{
			  "DockerId": "43481a6ce4842eec8fe72fc28500c6b52edcc0917f105b83379f88cac1ff3946",
			  "Name": "nginx-curl",
			  "DockerName": "ecs-nginx-5-nginx-curl-ccccb9f49db0dfe0d901",
			  "Image": "nrdlngr/nginx-curl",
			  "ImageID": "sha256:2e00ae64383cfc865ba0a2ba37f61b50a120d2d9378559dcd458dc0de47bc165",
			  "Labels": {
				"com.amazonaws.ecs.cluster": "default",
				"com.amazonaws.ecs.container-name": "nginx-curl",
				"com.amazonaws.ecs.task-arn": "arn:aws:ecs:us-east-2:012345678910:task/9781c248-0edd-4cdb-9a93-f63cb662a5d3",
				"com.amazonaws.ecs.task-definition-family": "nginx",
				"com.amazonaws.ecs.task-definition-version": "5"
			  },
			  "DesiredStatus": "RUNNING",
			  "KnownStatus": "RUNNING",
			  "Limits": {
				"CPU": 512,
				"Memory": 512
			  },
			  "CreatedAt": "2018-02-01T20:55:10.554941919Z",
			  "StartedAt": "2018-02-01T20:55:11.064236631Z",
			  "Type": "NORMAL",
			  "Networks": [
				{
				  "NetworkMode": "awsvpc",
				  "IPv4Addresses": [
					"10.0.2.106"
				  ]
				}
			  ]
			}
		  ],
		  "PullStartedAt": "2018-02-01T20:55:09.372495529Z",
		  "PullStoppedAt": "2018-02-01T20:55:10.552018345Z",
		  "AvailabilityZone": "us-east-2b"
		}`, stubTaskId))

	mock, err := config.GetRunningFargateTaskID("unittest")
	assert.Equal(t, stubTaskId, mock, "Task id should be returned correctly")
	assert.Nil(t, err, "err should return nil")
}

func TestGetConfig_DefaultRegionGroupsExist(t *testing.T) {

	config, err := config.GetConfig()

	require.NoError(t, err)
	require.NotEmpty(t, config.RegionGroups)
}
