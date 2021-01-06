package steps

import (
	"fmt"
	"strings"
	"testing"

	"github.optum.com/healthcarecloud/terrascale/pkg/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
)

var sut Stepper
var logger = logrus.NewEntry(logrus.New())
var DefaultStubAccountID = "1"

func TestNewExecution_ShouldSetFields(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	stubRegion := "region"
	stubRegionalDeployType := RegionalRegionDeployType

	stubStep := Step{
		Dir:  "stub",
		Name: "stubName",
		DeployConfig: config.Config{
			CSP:                       "stubCSP",
			DeploymentRing:            "stubDeploymentRing",
			Project:                   "stubProject",
			DryRun:                    true,
			TerrascaleTargetRegions:   []string{"stub"},
			UniqueExternalExecutionID: "stubExecutionID",
			MaxRetries:                3,
			MaxTestRetries:            2,
		},
		TrackName: "stubTrackName",
	}
	// act
	mock := NewExecution(stubStep, logger, fs, stubRegionalDeployType, stubRegion, map[string]map[string]string{})

	// assert
	require.Equal(t, stubStep.Dir, mock.Dir, "Dir should match stub value")
	require.Equal(t, stubStep.Name, mock.StepName, "Name should match stub value")
	require.Equal(t, stubRegion, mock.Region, "Region should match stub value")
	require.Equal(t, stubRegionalDeployType, mock.RegionDeployType, "RegionDeployType should match stub value")
	require.Equal(t, stubStep.DeployConfig.CSP, mock.CSP, "CSP should match stub value")
	require.Equal(t, stubStep.DeployConfig.DeploymentRing, mock.DeploymentRing, "DeploymentRing should match stub value")
	require.Equal(t, stubStep.DeployConfig.Project, mock.Project, "Project should match stub value")
	require.Equal(t, stubStep.DeployConfig.DryRun, mock.DryRun, "DryRun should match stub value")
	require.Equal(t, stubStep.TrackName, mock.TrackName, "TrackName should match stub value")
	require.Equal(t, stubStep.DeployConfig.UniqueExternalExecutionID, mock.UniqueExternalExecutionID, "UniqueExternalExecutionID should match stub value")
	require.Equal(t, stubStep.DeployConfig.TerrascaleTargetRegions, mock.RegionGroupRegions, "RegionGroupRegions should match stub value")
	require.Equal(t, stubStep.DeployConfig.MaxRetries, mock.MaxRetries, "MaxRetries should match stub value")
	require.Equal(t, stubStep.DeployConfig.MaxTestRetries, mock.MaxTestRetries, "MaxTestRetries should match stub value")

}

func TestGetBackendConfig_ShouldParseAssumeRoleCoreAccountIDMapCorrectly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "backend.tf", []byte(`
	terraform {
	  backend "s3" {
		key         = "/${var.terrascale_deployment_ring}-stub.tfstate"
		role_arn    = "arn:aws:iam::${var.core_account_ids_map.logging_bridge_aws}:role/OrganizationAccountAccessRole"
	  }
	}
	`), 0644)

	mockResult := GetBackendConfig(ExecutionConfig{
		Fs:     fs,
		Logger: logger,
		CoreAccounts: map[string]config.Account{
			"logging_bridge_aws": {ID: DefaultStubAccountID, CredsID: DefaultStubAccountID, CSP: DefaultStubAccountID, AccountOwnerLabel: DefaultStubAccountID},
		}}, ParseTFBackend)

	require.Equal(t, S3Backend, mockResult.Type)
	require.Equal(t, fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", DefaultStubAccountID), mockResult.Config["role_arn"])
}

func TestGetBackendConfig_ShouldInterpolateBucketField(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "backend.tf", []byte(`
	terraform {
	  backend "s3" {
		bucket      = "${var.terrascale_deployment_ring}-bucket"
	  }
	}
	`), 0644)

	mockResult := GetBackendConfig(ExecutionConfig{
		Fs:                                       fs,
		Logger:                                   logger,
		DeploymentRing:                           "fake",
		FeatureToggleDisableBackendDefaultBucket: true,
	}, ParseTFBackend)

	require.Equal(t, "fake-bucket", mockResult.Config["bucket"])
}

func TestGetBackendConfig_ShouldInterpolateResourceGroupNameField(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "backend.tf", []byte(`
	terraform {
	  backend "azurerm" {
		resource_group_name  = "rg-${var.terrascale_deployment_ring}"
	  }
	}
	`), 0644)

	mockResult := GetBackendConfig(ExecutionConfig{
		Fs:                                       fs,
		Logger:                                   logger,
		DeploymentRing:                           "fake",
		FeatureToggleDisableBackendDefaultBucket: true,
	}, ParseTFBackend)

	require.Equal(t, "rg-fake", mockResult.Config["resource_group_name"])
}

func TestGetBackendConfig_ShouldInterpolateStorageAccountNameField(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "backend.tf", []byte(`
	terraform {
	  backend "azurerm" {
		storage_account_name  = "st-${var.terrascale_deployment_ring}"
	  }
	}
	`), 0644)

	mockResult := GetBackendConfig(ExecutionConfig{
		Fs:                                       fs,
		Logger:                                   logger,
		DeploymentRing:                           "fake",
		FeatureToggleDisableBackendDefaultBucket: true,
	}, ParseTFBackend)

	require.Equal(t, "st-fake", mockResult.Config["storage_account_name"])
}

func TestGetBackendConfig_ShouldParseAssumeRoleStepCorrectly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "backend.tf", []byte(`
	terraform {
	  backend "s3" {
		key         = "/${var.terrascale_step}-stub.tfstate"
	  }
	}
	`), 0644)

	mockResult := GetBackendConfig(ExecutionConfig{
		Fs:       fs,
		Logger:   logger,
		StepName: "fakestep",
	}, ParseTFBackend)

	require.Equal(t, "fake-/fakestep-stub.tfstate/primary-", mockResult.Config["key"].(string))
}

func TestGetBackendConfig_ShouldHandleFeatureToggleDisableS3BackendKeyPrefixCorrectly(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "backend.tf", []byte(`
	terraform {
	  backend "s3" {
		key         = "noprefix"
	  }
	}
	`), 0644)

	mockResult := GetBackendConfig(ExecutionConfig{
		Fs:     fs,
		Logger: logger,
		CoreAccounts: map[string]config.Account{
			"logging_bridge_aws": {ID: DefaultStubAccountID, CredsID: DefaultStubAccountID, CSP: DefaultStubAccountID, AccountOwnerLabel: DefaultStubAccountID},
		},
		AccountID:                              "fun",
		FeatureToggleDisableS3BackendKeyPrefix: true,
	}, ParseTFBackend)

	require.True(t, strings.HasPrefix(mockResult.Config["key"].(string), "noprefix"), "%s should have no prefix appended when using FeatureToggleDisableS3BackendKeyPrefix", mockResult.Config["key"].(string))
}

func TestGetBackendConfig_ShouldReturnSameValueForKeyAsStepAsNoKey(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	_ = afero.WriteFile(fs, "backend.tf", []byte(`
	terraform {
	  backend "s3" {
		key         = "fakestep"
	  }
	}
	`), 0644)

	mockResult := GetBackendConfig(ExecutionConfig{
		Fs:        fs,
		Logger:    logger,
		AccountID: "fun",
		StepName:  "fakestep",
	}, ParseTFBackend)

	fs2 := afero.NewMemMapFs()

	_ = afero.WriteFile(fs2, "backend.tf", []byte(`
	terraform {
	  backend "s3" { }
	}
	`), 0644)

	mockResult2 := GetBackendConfig(ExecutionConfig{
		Fs:        fs,
		Logger:    logger,
		AccountID: "fun",
		StepName:  "fakestep",
	}, ParseTFBackend)

	require.Equal(t, mockResult.Config["key"].(string), mockResult2.Config["key"].(string))
}

func TestHandleOverrides_ShouldSetFields(t *testing.T) {
	var overrideSrc, overrideDst string

	CopyFile = func(src, dst string) (err error) {
		overrideSrc = src
		overrideDst = dst
		return nil
	}

	// act
	handleOverride(logger, "src", "test.tf")

	// assert
	require.Equal(t, "src/override/test.tf", overrideSrc, "src should be set to overrides directory")
	require.Equal(t, "src/test.tf", overrideDst, "src should be set to execution directory")
}

func TestHandleDeployOverrides_ShouldSetFields(t *testing.T) {
	var deploySrc, deployDst, deployRingSrc, deployRingDst string

	CopyFile = func(src, dst string) (err error) {
		if deploySrc == "" {
			deploySrc = src
			deployDst = dst
		} else {
			deployRingSrc = src
			deployRingDst = dst
		}
		return nil
	}

	// act
	HandleDeployOverrides(logger, "src", "test")

	// assert
	require.Equal(t, "src/override/override.tf", deploySrc, "src should be set to overrides directory")
	require.Equal(t, "src/override.tf", deployDst, "src should be set to execution directory")
	require.Equal(t, "src/override/ring_test_override.tf", deployRingSrc, "src should be set to overrides directory")
	require.Equal(t, "src/ring_test_override.tf", deployRingDst, "src should be set to execution directory")
}

func TestHandleDestroyOverrides_ShouldSetFields(t *testing.T) {
	var destroySrc, destroyDst, destroyRingSrc, destroyRingDst string

	CopyFile = func(src, dst string) (err error) {
		if destroySrc == "" {
			destroySrc = src
			destroyDst = dst
		} else {
			destroyRingSrc = src
			destroyRingDst = dst
		}
		return nil
	}

	// act
	HandleDestroyOverrides(logger, "src", "test")

	// assert
	require.Equal(t, "src/override/destroy_override.tf", destroySrc, "src should be set to overrides directory")
	require.Equal(t, "src/destroy_override.tf", destroyDst, "src should be set to execution directory")
	require.Equal(t, "src/override/destroy_ring_test_override.tf", destroyRingSrc, "src should be set to overrides directory")
	require.Equal(t, "src/destroy_ring_test_override.tf", destroyRingDst, "src should be set to execution directory")
}

func TestExecuteStepDestroy_ShouldSkipWhenRegionNotInExecuteWhen(t *testing.T) {
	// act
	execDestroy := TerraformStepper{}.ExecuteStepDestroy(ExecutionConfig{
		TerrascaleConfig: TerrascaleConfig{
			ExecuteWhen: TerrascaleConfigExecuteWhen{
				RegionIn: []string{"stub-region"},
			}},
		Region: "not-stub-region",
		Logger: logger,
	})

	// assert
	require.Equal(t, Na, execDestroy.Status, "Status should be skipped")
}

func TestExecuteStep_ShouldSkipWhenRegionNotInExecuteWhen(t *testing.T) {
	// act
	exec := TerraformStepper{}.ExecuteStep(ExecutionConfig{
		TerrascaleConfig: TerrascaleConfig{
			ExecuteWhen: TerrascaleConfigExecuteWhen{
				RegionIn: []string{"stub-region"},
			}},
		Region: "not-stub-region",
		Logger: logger,
	})

	// assert
	require.Equal(t, Na, exec.Status, "Status should be skipped")
}

func TestExecuteStep_ShouldExecuteWhenExecuteWhenUndefined(t *testing.T) {
	executed := 0
	executeTerraformInDir = func(exec ExecutionConfig, destroy bool) (output StepOutput) {
		executed++
		return
	}
	// act
	_ = TerraformStepper{}.ExecuteStep(ExecutionConfig{
		Region: "not-stub-region",
		Logger: logger,
	})

	// assert
	require.Equal(t, 1, executed, "Step should have executed")
}

func TestGetBackendConfig_ShouldCorrectlyHandleParsedBackend2(t *testing.T) {
	t.Parallel()

	getBackendTests := map[string]struct {
		stubParsedBackend TerraformBackend
		environment       string
		region            string
		regionType        RegionDeployType
		expect            string
		expectNil         bool
		namespace         string
	}{
		"ShouldJoinParsedKeyWithNamespace": {
			stubParsedBackend: TerraformBackend{
				Key:  "key",
				Type: S3Backend,
			},
			environment: "prod",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			namespace:   "",
			expect:      "fake-accountID/key",
		},
		"ShouldSanitizeDoubleSlash": {
			stubParsedBackend: TerraformBackend{
				Key:  "/key",
				Type: S3Backend,
			},
			environment: "prod",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			expect:      "fake-accountID/key",
		},
		"ShouldNamespaceStateFileAndNotPath": {
			stubParsedBackend: TerraformBackend{
				Key:  "/directory/state",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			namespace:   "namespace",
			expect:      "fake-accountID/directory/namespace-state",
		},
		"ShouldNotNamespaceStateFileWhenNamespaceIsEmpty": {
			stubParsedBackend: TerraformBackend{
				Key:  "/directory/state",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			namespace:   "",
			expect:      "fake-accountID/directory/state",
		},
		"ShouldIncludeRegionWhenRegional": {
			stubParsedBackend: TerraformBackend{
				Key:  "/directory/state",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "us-east-2",
			regionType:  RegionalRegionDeployType,
			namespace:   "namespace",
			expect:      "fake-accountID/directory/namespace-state/regional-us-east-2",
		},
		"ShouldNotIncludeRegionWhenPrimaryAndUsEast1": {
			stubParsedBackend: TerraformBackend{
				Key:  "/directory/state",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			namespace:   "namespace",
			expect:      "fake-accountID/directory/namespace-state",
		},
		"ShouldNotIncludeRegionWhenPrimaryAndNotUsEast1": {
			stubParsedBackend: TerraformBackend{
				Key:  "/directory/state",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "us-east-2",
			regionType:  PrimaryRegionDeployType,
			namespace:   "namespace",
			expect:      "fake-accountID/directory/namespace-state/primary-us-east-2",
		},
		"ShouldIncludeRegionWhenRegionalAndUsEast1": {
			stubParsedBackend: TerraformBackend{
				Key:  "/directory/state",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "us-east-1",
			regionType:  RegionalRegionDeployType,
			namespace:   "namespace",
			expect:      "fake-accountID/directory/namespace-state/regional-us-east-1",
		},
		"ShouldVarSubstituteTerrascaleDeploymentRing": {
			stubParsedBackend: TerraformBackend{
				Key:  "/${var.terrascale_deployment_ring}/key",
				Type: S3Backend,
			},
			environment: "prod",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			expect:      "fake-accountID/deploymentring/key",
		},
		"ShouldNamespaceWhenPRAndNoDeclaredBackendKeyAndUsEast1": {
			stubParsedBackend: TerraformBackend{
				Key:  "",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			namespace:   "namespace",
			expect:      "fake-accountID/namespace-step1_deploy.tfstate",
		},
		"ShouldNamespaceWhenPRAndNoDeclaredBackendKeyAndNotUsEast1": {
			stubParsedBackend: TerraformBackend{
				Key:  "",
				Type: S3Backend,
			},
			environment: "pr",
			region:      "centralus",
			regionType:  PrimaryRegionDeployType,
			namespace:   "namespace",
			expect:      "fake-accountID/namespace-step1_deploy/primary-centralus.tfstate",
		},
		"ShouldIncludeRegionalWhenNotUsEast1AndNotNamespaceInProd": {
			stubParsedBackend: TerraformBackend{
				Key:  "",
				Type: S3Backend,
			},
			environment: "prod",
			region:      "eastus",
			regionType:  RegionalRegionDeployType,
			expect:      "fake-accountID/step1_deploy/regional-eastus.tfstate",
		},
		"ShouldCorrectlyParseLocalBack": {
			stubParsedBackend: TerraformBackend{
				Key:  "",
				Type: LocalBackend,
			},
			environment: "prod",
			region:      "eastus",
			regionType:  RegionalRegionDeployType,
			expectNil:   true,
		},
		"ShouldCorrectlyParseGCSBackend": {
			stubParsedBackend: TerraformBackend{
				GCSPrefix: "",
				Type:      GCSBackend,
			},
			environment: "prod",
			region:      "us-central1",
			regionType:  PrimaryRegionDeployType,
			expectNil:   true,
		},
	}

	fs := afero.NewMemMapFs()

	for name, tc := range getBackendTests {
		t.Run(name, func(t *testing.T) {
			exec := ExecutionConfig{
				RegionDeployType:           tc.regionType,
				Region:                     tc.region,
				Logger:                     logger,
				Fs:                         fs,
				DefaultStepOutputVariables: map[string]map[string]string{},
				CredsID:                    "creds",
				Environment:                tc.environment,
				Namespace:                  tc.namespace,
				AccountID:                  "accountID",
				TerrascaleTargetAccountID:  "accountID",
				DeploymentRing:             "deploymentring",
				RegionGroup:                "us",
				Dir:                        "/tracks/step1_deploy",
				StepName:                   "step1_deploy",
			}

			stubParseTFBackend := func(fs afero.Fs, log *logrus.Entry, file string) TerraformBackend {
				return tc.stubParsedBackend
			}
			received := GetBackendConfig(exec, stubParseTFBackend)

			if tc.expectNil {
				require.Nil(t, received.Config["key"])
			} else {
				require.Equal(t, tc.expect, received.Config["key"])
			}
			require.Equal(t, tc.stubParsedBackend.Type, received.Type)
		})
	}
}

func TestGetBackendConfig_ShouldCorrectlyHandleParseGCSBackend(t *testing.T) {
	t.Parallel()

	getBackendTests := map[string]struct {
		stubParsedBackend TerraformBackend
		environment       string
		region            string
		regionType        RegionDeployType
		expectBucket      string
		expectPrefix      string
		namespace         string
	}{
		"ShouldCorrectlyParseGCSBackend": {
			stubParsedBackend: TerraformBackend{
				GCSBucket: "test-${var.environment}-tfstate",
				GCSPrefix: "test/${var.terrascale_deployment_ring}/${var.terrascale_region_deploy_type}/${var.region}/${local.namespace-}test.tfstate",
				Type:      GCSBackend,
			},
			environment:  "prod",
			region:       "us-central1",
			regionType:   PrimaryRegionDeployType,
			expectBucket: "test-prod-tfstate",
			expectPrefix: "test/deploymentring/primary/us-central1/test.tfstate",
		},
	}

	fs := afero.NewMemMapFs()

	for name, tc := range getBackendTests {
		t.Run(name, func(t *testing.T) {
			exec := ExecutionConfig{
				RegionDeployType:           tc.regionType,
				Region:                     tc.region,
				Logger:                     logger,
				Fs:                         fs,
				DefaultStepOutputVariables: map[string]map[string]string{},
				CredsID:                    "creds",
				Environment:                tc.environment,
				Namespace:                  tc.namespace,
				AccountID:                  "accountID",
				DeploymentRing:             "deploymentring",
				RegionGroup:                "us",
				Dir:                        "/tracks/step1_deploy",
				StepName:                   "step1_deploy",
			}

			stubParseTFBackend := func(fs afero.Fs, log *logrus.Entry, file string) TerraformBackend {
				return tc.stubParsedBackend
			}
			received := GetBackendConfig(exec, stubParseTFBackend)

			require.Equal(t, tc.expectBucket, received.Config["bucket"])
			require.Equal(t, tc.expectPrefix, received.Config["prefix"])
			require.Equal(t, tc.stubParsedBackend.Type, received.Type)
		})
	}
}

func TestGetBackendConfig_ShouldCorrectlyHandleParsedBackendWithFeatureDisables(t *testing.T) {
	t.Parallel()

	getBackendTests := map[string]struct {
		stubParsedBackend TerraformBackend
		environment       string
		region            string
		regionType        RegionDeployType
		expect            string
		expectNil         bool
		namespace         string
	}{
		"ShouldVarSubstituteTerrascaleDeploymentRingAndCoreAccountIds": {
			stubParsedBackend: TerraformBackend{
				Key:  "fake-${var.core_account_ids_map.gcp_core_project}/${var.terrascale_deployment_ring}.tfstate",
				Type: S3Backend,
			},
			environment: "prod",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			expect:      "fake-projectId/deploymentring.tfstate",
		},
		"ShouldSubstituteAllInstancesOfCoreAccountIdsMaps": {
			stubParsedBackend: TerraformBackend{
				Key:  "fake-${var.core_account_ids_map.logging_bridge_gcp}/${var.core_account_ids_map.gcp_core_project}/${var.terrascale_deployment_ring}.tfstate",
				Type: S3Backend,
			},
			environment: "prod",
			region:      "us-east-1",
			regionType:  PrimaryRegionDeployType,
			expect:      "fake-projectId2/projectId/deploymentring.tfstate",
		},
	}

	fs := afero.NewMemMapFs()

	for name, tc := range getBackendTests {
		t.Run(name, func(t *testing.T) {
			exec := ExecutionConfig{
				RegionDeployType:                         tc.regionType,
				Region:                                   tc.region,
				Logger:                                   logger,
				Fs:                                       fs,
				DefaultStepOutputVariables:               map[string]map[string]string{},
				CredsID:                                  "creds",
				Environment:                              tc.environment,
				Namespace:                                tc.namespace,
				AccountID:                                "accountID",
				TerrascaleTargetAccountID:                "accountID",
				DeploymentRing:                           "deploymentring",
				RegionGroup:                              "us",
				Dir:                                      "/tracks/step1_deploy",
				StepName:                                 "step1_deploy",
				FeatureToggleDisableS3BackendKeyPrefix:   true,
				FeatureToggleDisableBackendDefaultBucket: true,
				CoreAccounts: map[string]config.Account{
					"gcp_core_project": {
						ID: "projectId",
					},
					"logging_bridge_gcp": {
						ID: "projectId2",
					},
				},
			}

			stubParseTFBackend := func(fs afero.Fs, log *logrus.Entry, file string) TerraformBackend {
				return tc.stubParsedBackend
			}
			received := GetBackendConfig(exec, stubParseTFBackend)

			if tc.expectNil {
				require.Nil(t, received.Config["key"])
			} else {
				require.Equal(t, tc.expect, received.Config["key"])
			}
			require.Equal(t, tc.stubParsedBackend.Type, received.Type)
		})
	}
}

func TestGetBackendConfigWithTerrascaleTargetAccountID_ShouldHandleSettingCorrectAccountDirectory2(t *testing.T) {
	t.Parallel()

	getBackendTests := map[string]struct {
		accountID                 string
		terrascaleTargetAccountID string
		expectedAccountID         string
		message                   string
	}{
		"ShouldSetCorrectlyWithMatchingValues": {
			accountID:                 "12",
			terrascaleTargetAccountID: "12",
			expectedAccountID:         "12",
			message:                   "Should set correctly when both values the same",
		},
		"ShouldPreferTerrascaleTargetAccountIDWithDifferingValues": {
			accountID:                 "13",
			terrascaleTargetAccountID: "12",
			expectedAccountID:         "12",
			message:                   "Should prefer terrascale target account id when both values set and differ",
		},
		"ShouldPreferAccountIDWhenTerrascaleTargetAccountIDNotSet": {
			accountID:                 "12",
			terrascaleTargetAccountID: "",
			expectedAccountID:         "12",
			message:                   "Should account id when terrascale target account id is not set",
		},
	}

	fs := afero.NewMemMapFs()

	for name, tc := range getBackendTests {
		t.Run(name, func(t *testing.T) {
			stubBackendParserResponse := TerraformBackend{
				Key:  "key",
				Type: S3Backend,
			}
			stubParseTFBackend := func(fs afero.Fs, log *logrus.Entry, file string) TerraformBackend {
				return stubBackendParserResponse
			}

			exec := ExecutionConfig{
				Region:                     "us-east-1",
				RegionDeployType:           PrimaryRegionDeployType,
				Logger:                     logger,
				Fs:                         fs,
				CredsID:                    "creds",
				Environment:                "environment",
				AccountID:                  tc.accountID,
				TerrascaleTargetAccountID:  tc.terrascaleTargetAccountID,
				StepName:                   "step1_deploy",
				Dir:                        "/tracks/step1_deploy",
				DefaultStepOutputVariables: map[string]map[string]string{},
			}

			// act
			received := GetBackendConfig(exec, stubParseTFBackend)

			// assert
			require.Equal(t, fmt.Sprintf("fake-%s/%s", tc.expectedAccountID, stubBackendParserResponse.Key), received.Config["key"])
			require.Equal(t, stubBackendParserResponse.Type, exec.TFBackend.Type)
		})
	}
}
