package steps_test

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
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
		key         = "/aws/core/logging/${var.terrascale_deployment_ring}-consumeraas_aws.tfstate"
		role_arn    = "stubrolearn"
	  }
	}
	`), 0644)

	mockResult := steps.ParseTFBackend(fs, logger, "testbackend.tf")

	require.Equal(t, steps.S3Backend, mockResult.Type)
	require.Equal(t, "stubrolearn", mockResult.S3RoleArn)
}

func TestParseProvider_ShouldParseAssumeRoleCorrectly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	stubAccountIDKey := "logging_final_destination,"
	mockAccount := config.Account{
		CredsID: "POC",
		ID:      "123456789012",
		CSP:     "AWS",
	}
	accountIds := map[string]config.Account{
		stubAccountIDKey: mockAccount,
	}
	_ = afero.WriteFile(fs, "providers.tf", []byte(fmt.Sprintf(`
	provider "aws" {
	  version             = "2.17.0"
	  region              = var.aws_region
	  allowed_account_ids = [var.final_destination_account_id]
	  assume_role {
		role_arn = "arn:aws:iam::%s:role/OrganizationAccountAccessRole"
	  }
	}
	`, fmt.Sprintf("${var.core_account_ids_map.%v}", stubAccountIDKey))), 0644)

	mockResult, err := steps.ParseTFProvider(fs, logger, ".", accountIds)

	//require.Equal(t, fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", mockAccountID), mockResult.AssumeRoleRoleArn)
	require.Nil(t, err, "Err should be nil")
	require.Equal(t, mockAccount.ID, mockResult.AssumeRoleAccount.ID)
	require.Equal(t, mockAccount.CredsID, mockResult.AssumeRoleAccount.CredsID)
	require.Equal(t, mockAccount.CSP, mockResult.AssumeRoleAccount.CSP)
	require.Equal(t, steps.AWSProvider, mockResult.Type, "The provider types should match")
}

func TestParseProvider_ShouldParseAssumeRoleCorrectlyWithAzurerm(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	stubAccountIDKey := "logging_final_destination,"
	mockAccount := config.Account{
		CredsID: "POC",
		ID:      "123456789012",
	}
	accountIds := map[string]config.Account{
		stubAccountIDKey: mockAccount,
	}
	_ = afero.WriteFile(fs, "providers.tf", []byte(`
	provider "azurerm" {
	  version = "1.33.1"
	}
	`), 0644)

	mockResult, err := steps.ParseTFProvider(fs, logger, ".", accountIds)

	//require.Equal(t, fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", mockAccountID), mockResult.AssumeRoleRoleArn)
	require.Nil(t, err, "Err should be nil")
	require.Equal(t, "", mockResult.AssumeRoleAccount.ID)
	require.Equal(t, steps.AzurermProvider, mockResult.Type, "The provider types should match")
}

func TestParseProvider_ShouldCorrectlyReturnErrWhenNotSupported(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	stubAccountIDKey := "final_destination_account_id"
	mockAccount := config.Account{
		CredsID: "POC",
		ID:      "123456789012",
	}
	accountIds := map[string]config.Account{
		stubAccountIDKey: mockAccount,
	}
	_ = afero.WriteFile(fs, "providers.tf", []byte(fmt.Sprintf(`
	provider "aws" {
	  version             = "2.17.0"
	  region              = var.aws_region
	  allowed_account_ids = [var.final_destination_account_id]
	  assume_role {
		role_arn = "arn:aws:iam::${var.%s}:role/OrganizationAccountAccessRole"
	  }
	}
	`, stubAccountIDKey)), 0644)

	_, err := steps.ParseTFProvider(fs, logger, ".", accountIds)

	//require.Equal(t, fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", mockAccountID), mockResult.AssumeRoleRoleArn)
	require.NotNil(t, err, "Err should be returned with unsupported configuration")
}

func TestParseProvider_ShouldNotThrowErrAndReturnDefaultProviderWhenFileNotPresent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	mock, err := steps.ParseTFProvider(fs, logger, ".", map[string]config.Account{})

	//require.Equal(t, fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", mockAccountID), mockResult.AssumeRoleRoleArn)
	require.Nil(t, err, "Err should be nil with no provider file")
	require.False(t, mock.AccountOverridden, "Provider AccountOverridden value should be false with no provider file")
}

func TestTFProviderTypeToString(t *testing.T) {
	tests := []struct {
		provider       steps.TFProviderType
		expectedString string
	}{
		{
			provider:       steps.AWSProvider,
			expectedString: "aws",
		},
		{
			provider:       steps.AzurermProvider,
			expectedString: "azurerm",
		},
	}

	for _, tc := range tests {
		result := tc.provider.String()
		require.Equal(t, tc.expectedString, result, "The string should match")
	}
}

func TestStringToProviderType(t *testing.T) {
	tests := []struct {
		s                string
		expectedProvider steps.TFProviderType
		errorExists      bool
	}{
		{
			s:                "aws",
			expectedProvider: steps.AWSProvider,
			errorExists:      false,
		},
		{
			s:                "azurerm",
			expectedProvider: steps.AzurermProvider,
			errorExists:      false,
		},
		{
			s:                "doesnotexist",
			expectedProvider: steps.UnknownProvider,
			errorExists:      true,
		},
	}

	for _, tc := range tests {
		result, err := steps.StringToProviderType(tc.s)
		require.Equal(t, tc.expectedProvider, result, "The providers should match")
		require.Equal(t, tc.errorExists, err != nil, "The error result should match the expected")
	}
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

func TestParseTFProvider_ShouldCorrectlyParseAzureSubscriptionIDInProvider(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	stubAccountIDKey := "tenant_core_azu"
	mockAccount := config.Account{
		CredsID: "UHG",
		ID:      "bc85422d-1b21-4020-ae83-712e6fba0fc0",
		CSP:     "AZU",
	}
	accountIds := map[string]config.Account{
		stubAccountIDKey: mockAccount,
	}
	_ = afero.WriteFile(fs, "providers.tf", []byte(fmt.Sprintf(`
	provider "azurerm" {
	  version             = "1.35.0"
	  subscription_id 		= var.core_account_ids_map.tenant_core_azu
	}
	`)), 0644)

	mockResult, err := steps.ParseTFProvider(fs, logger, ".", accountIds)

	require.Nil(t, err, "Err should be nil")
	require.Equal(t, mockAccount.ID, mockResult.AssumeRoleAccount.ID)
	require.Equal(t, mockAccount.CredsID, mockResult.AssumeRoleAccount.CredsID)
	require.Equal(t, mockAccount.CSP, mockResult.AssumeRoleAccount.CSP)
	require.Equal(t, steps.AzurermProvider, mockResult.Type, "The provider types should match")
}
