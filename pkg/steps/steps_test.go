package steps_test

import (
	"flag"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.optum.com/healthcarecloud/terrascale/pkg/steps"

	"github.com/spf13/afero"
)

var DefaultStubAccountID = "1"
var StubVersion = "v0.0.5"
var logger *logrus.Entry
var sut steps.Stepper

func TestMain(m *testing.M) {

	logs := logrus.New()
	logs.SetLevel(logrus.WarnLevel)
	logger = logrus.NewEntry(logs)

	sut = steps.TerraformStepper{}

	flag.Parse()
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}

func TestParseBackend_ShouldParseS3Correctly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "testbackend.tf", []byte(`
	terraform {
	  backend "s3" {}
	}
	`), 0644)

	mockResult := steps.ParseTFBackend(fs, logger, "testbackend.tf")

	require.Equal(t, steps.S3Backend, mockResult.Type)
	require.Equal(t, "", mockResult.Key)
}

func TestParseBackend_ShouldParseS3WithKeyCorrectly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "testbackend.tf", []byte(`
	terraform {
	  backend "s3" {
	    key = "bedrock-enduser-iam.tfstate"	
	  }
	}
	`), 0644)

	mockResult := steps.ParseTFBackend(fs, logger, "testbackend.tf")

	require.Equal(t, steps.S3Backend, mockResult.Type)
	require.Equal(t, "bedrock-enduser-iam.tfstate", mockResult.Key)
}

func TestParseBackend_ShouldParseS3WithMalformedKeyCorrectly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "testbackend.tf", []byte(`
	terraform {
	  backend "s3" {
	    key="bedrock-enduser-iam.tfstate"	
	  }
	}
	`), 0644)

	mockResult := steps.ParseTFBackend(fs, logger, "testbackend.tf")

	require.Equal(t, steps.S3Backend, mockResult.Type)
	require.Equal(t, "bedrock-enduser-iam.tfstate", mockResult.Key)
}

func TestParseBackend_ShouldParseLocalCorrectly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "testbackend.tf", []byte(`
	terraform {
	  backend "local" {}
	}
	`), 0644)

	mockResult := steps.ParseTFBackend(fs, logger, "testbackend.tf")

	require.Equal(t, steps.LocalBackend, mockResult.Type)
	require.Equal(t, "", mockResult.Key)
}

func TestParseBackend_ShouldParseRoleArnWhenSet(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "testbackend.tf", []byte(`
	terraform {
	  backend "s3" {
		key         = "/aws/core/logging/${var.terrascale_deployment_ring}-stub.tfstate"
		role_arn    = "stubrolearn"
	  }
	}
	`), 0644)

	mockResult := steps.ParseTFBackend(fs, logger, "testbackend.tf")

	require.Equal(t, steps.S3Backend, mockResult.Type)
	require.Equal(t, "stubrolearn", mockResult.S3RoleArn)
}

func TestTFBackendTypeToString(t *testing.T) {
	tests := []struct {
		backend        steps.TFBackendType
		expectedString string
	}{
		{
			backend:        steps.S3Backend,
			expectedString: "s3",
		},
		{
			backend:        steps.LocalBackend,
			expectedString: "local",
		},
	}

	for _, tc := range tests {
		result := tc.backend.String()
		require.Equal(t, tc.expectedString, result, "The string should match")
	}
}

func TestStringToBackendType(t *testing.T) {
	tests := []struct {
		s               string
		expectedBackend steps.TFBackendType
		errorExists     bool
	}{
		{
			s:               "s3",
			expectedBackend: steps.S3Backend,
			errorExists:     false,
		},
		{
			s:               "local",
			expectedBackend: steps.LocalBackend,
			errorExists:     false,
		},
		{
			s:               "doesnotexist",
			expectedBackend: steps.UnknownBackend,
			errorExists:     true,
		},
	}

	for _, tc := range tests {
		result, err := steps.StringToBackendType(tc.s)
		require.Equal(t, tc.expectedBackend, result, "The backends should match")
		require.Equal(t, tc.errorExists, err != nil, "The error result should match the expected")
	}
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
