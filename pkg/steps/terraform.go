//go:generate mockgen -destination ../../mocks/mock_steps.go -package=mocks github.optum.com/healthcarecloud/terrascale/pkg/steps StepperFactory,Stepper

package steps

import (
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"path/filepath"
	"regexp"
	"strings"
)

// TFProviderType represents a Terraform provider type
type TFProviderType int

const (
	// AWSProvider is the AWS TF provider
	AWSProvider TFProviderType = iota
	// AzurermProvider is the Azurerm TF provider
	AzurermProvider
	// UnknownProvider represents a provider that could not be determined
	UnknownProvider
)

func (p TFProviderType) String() string {
	return [...]string{"aws", "azurerm", "unknown"}[p]
}

// StringToProviderType converts a string to a ProviderType
func StringToProviderType(s string) (TFProviderType, error) {
	providers := map[string]TFProviderType{
		"aws":     AWSProvider,
		"azurerm": AzurermProvider,
	}

	val, exists := providers[s]
	if !exists {
		return UnknownProvider, errors.New("Invalid provider string")
	}
	return val, nil
}

// TFBackendType represents a Terraform backend type
type TFBackendType int

const (
	// S3Backend backend
	S3Backend TFBackendType = iota
	// Azure Storage Account backend
	AzureStorageAccount
	// Google Cloud Storage backend
	GCSBackend
	// LocalBackend backend
	LocalBackend
	// UnknownBackend represents an unknown backend
	UnknownBackend
)

func (b TFBackendType) String() string {
	return [...]string{"s3", "azurerm", "gcs", "local", "unknown"}[b]
}

// StringToBackendType converts a string to a ProviderType
func StringToBackendType(s string) (TFBackendType, error) {
	backends := map[string]TFBackendType{
		"s3":      S3Backend,
		"azurerm": AzureStorageAccount,
		"gcs":     GCSBackend,
		"local":   LocalBackend,
	}

	val, exists := backends[s]
	if !exists {
		return UnknownBackend, errors.New("Invalid backend string")
	}
	return val, nil
}

// TerraformProvider is a structure that represents a terraform providers file
type TerraformProvider struct {
	Type              TFProviderType
	AccountOverridden bool
	AssumeRoleAccount config.Account
}

// TerraformBackend is a structure that represents a terraform backend file
type TerraformBackend struct {
	Type      TFBackendType
	Key       string
	S3RoleArn string
	S3Bucket  string
	AZUResourceGroupName      string
	AZUStorageAccountName string
	GCSBucket string
	GCSPrefix string
	Config    map[string]interface{}
}

// TFBackendParser is a function type that handles parsing a backend.tf file
type TFBackendParser func(fs afero.Fs, log *logrus.Entry, file string) (backend TerraformBackend)

func getStateFile(tfStateName string, namespace string, ring string, environment string, region string, regionType RegionDeployType) string {
	var namespacedStateFile = tfStateName

	if namespace != "" {
		namespacedStateFile = fmt.Sprintf("%s-%s", namespace, namespacedStateFile)
	}

	if region != "us-east-1" || regionType == RegionalRegionDeployType {
		regionNamespace := fmt.Sprintf("%s-%s", regionType.String(), region)
		namespacedStateFile = filepath.Join(namespacedStateFile, regionNamespace)
	}

	return namespacedStateFile
}

// ParseTFBackend parses a backend.tf file
func ParseTFBackend(fs afero.Fs, log *logrus.Entry, file string) (backend TerraformBackend) {
	backendString := "backend \""
	// read the whole file at once
	b, err := afero.ReadFile(fs, file)
	if err != nil {
		log.WithError(err).Error(err)
		backend.Type = LocalBackend
		return
	}

	s := string(b)

	// backend Type
	backendType, backendErr := StringToBackendType(strings.Split(strings.Split(s, backendString)[1], "\"")[0])
	if backendErr != nil {
		log.WithError(backendErr).Fatal("Invalid backend type")
	}
	backend.Type = backendType

	// Key
	r, _ := regexp.Compile(`key\s*=\s*"(.+)"`)
	substring := r.FindStringSubmatch(s)

	if len(substring) > 0 {
		backend.Key = substring[1]
	}

	// RoleArn
	rRegex, _ := regexp.Compile(`role_arn\s*=\s*"(.+)"`)
	roleMatch := rRegex.FindStringSubmatch(s)

	if len(roleMatch) > 0 {
		backend.S3RoleArn = roleMatch[1]
	}

	// Bucket
	bRegex, _ := regexp.Compile(`bucket\s*=\s*"(.+)"`)
	bucketMatch := bRegex.FindStringSubmatch(s)

	if len(bucketMatch) > 0 {
		if backendType == S3Backend {
			backend.S3Bucket = bucketMatch[1]
		} else if backendType == GCSBackend {
			backend.GCSBucket = bucketMatch[1]
		}
	}

	// Prefix
	pRegex, _ := regexp.Compile(`prefix\s*=\s*"(.+)"`)
	prefixMatch := pRegex.FindStringSubmatch(s)

	if backendType == GCSBackend && len(prefixMatch) > 0 {
		backend.GCSPrefix = prefixMatch[1]
	}

	// Resource group (Azure)
	rgRegex, _ := regexp.Compile(`resource_group_name\s*=\s*"(.+)"`)
	resourceGroupMatch := rgRegex.FindStringSubmatch(s)

	if len(resourceGroupMatch) > 0 {
		backend.AZUResourceGroupName = resourceGroupMatch[1]
	}

	// Storage account (Azure)
	stRegex, _ := regexp.Compile(`storage_account_name\s*=\s*"(.+)"`)
	storageAccountMatch := stRegex.FindStringSubmatch(s)

	if len(storageAccountMatch) > 0 {
		backend.AZUStorageAccountName = storageAccountMatch[1]
	}

	return
}

// ParseTFProvider parses a providers.tf file
func ParseTFProvider(fs afero.Fs, logger *logrus.Entry, dir string, accountIds map[string]config.Account) (provider TerraformProvider, err error) {

	file := fmt.Sprintf("%s/providers.tf", dir)
	// check if file exists
	exists, err := afero.Exists(fs, file)
	if !exists {
		if err != nil {
			logger.WithError(err).Error(err)
		}
		return
	}

	// read the whole file at once
	b, err := afero.ReadFile(fs, file)
	if err != nil {
		logger.WithError(err).Error(err)
		return
	}

	s := string(b)

	// Determine provider type
	providerSplit := strings.Split(s, " ")
	providerString := strings.Replace(providerSplit[1], "\"", "", -1)
	providerType, _ := StringToProviderType(providerString)
	provider.Type = providerType

	logger.Debugf("Parsed provider type: %s", provider.Type)

	if provider.Type == AWSProvider {
		var assumeRoleRoleArn string

		// TODO: greatly improve this. likely by leveraging gohcl or terraform packages to properly handle hcl variables and parsing?
		// also consider attempting assumerole into provided arn (rather than extract account id and use OrganizationAccountAccessRole)
		r, _ := regexp.Compile(`role_arn\s*=\s*"(.+)"`)

		substring := r.FindStringSubmatch(s)

		if len(substring) > 0 {
			assumeRoleRoleArn = substring[1]
		}

		// if assume role arn is set, extract out account id for proper logging
		if assumeRoleRoleArn != "" {
			provider.AccountOverridden = true

			// assume_role_arn will match arn:aws:iam::123456789012:role/S3Access
			accountID := strings.Split(strings.Split(assumeRoleRoleArn, "::")[1], ":role")[0]

			logger.Debug(fmt.Sprintf("Parse Provider Assume Role: %v", accountID))

			// check for variable arn:aws:iam::${var.core_account_ids_map.logging_final_destination}:role/OrganizationAccountAccessRole
			checkForCoreAccount := strings.Split(accountID, "var.core_account_ids_map.")
			checkForTargetAccount := strings.Contains(accountID, "var.terrascale_target_account_id")

			logger.Debug(fmt.Sprintf("Parse Provider Assume Role: %v", strings.Join(checkForCoreAccount, ", ")))

			var accountIDKey string
			if len(checkForCoreAccount) > 1 {
				// parse out variable key ${var.core_account_ids_map.logging_final_destination}
				accountIDKey = strings.Split(checkForCoreAccount[1], "}")[0]

			} else if checkForTargetAccount {
				accountIDKey = "terrascale_target_account_id"
			}

			logger.Debug(fmt.Sprintf("Parse Provider Assume Role: %v", accountIDKey))

			if val, ok := accountIds[accountIDKey]; ok {
				provider.AssumeRoleAccount = val
			} else {
				logger.Warnf(`Did not find match for variable "%v" while parsing provider assume role. Possible options include: %v`, accountIDKey, KeysString(accountIds))
			}

			// TODO(GAIA_FEATURE_DISABLE_CREDS): remove this error return after ensuring all projects have this feature toggle as true
			if len(provider.AssumeRoleAccount.CredsID) == 0 {
				return provider, errors.New("Unable to correctly set AssumeRoleAccount from provider.tf assume role override")
			}
		}
	} else if provider.Type == AzurermProvider {
		var subscriptionID string

		r, _ := regexp.Compile(`subscription_id\s*=\s*(.+)`)
		substring := r.FindStringSubmatch(s)

		if len(substring) > 0 {
			subscriptionID = substring[1]
		}

		if subscriptionID != "" {
			provider.AccountOverridden = true

			// subscription_id = var.core_account_ids_map.core_azu
			checkForVariable := strings.Split(subscriptionID, "var.core_account_ids_map.")

			if len(checkForVariable) > 0 {
				accountIDKey := checkForVariable[1]

				if val, ok := accountIds[accountIDKey]; ok {
					provider.AssumeRoleAccount = val
				} else {
					logger.Errorf(`Did not find match for variable "%v" while parsing subscription id. Possible options include: %v`, accountIDKey, KeysString(accountIds))
				}
			} else {
				return provider, errors.New(fmt.Sprintf("Unsupported subscription id configuration: %v. Unable to find core_account_ids_map.", subscriptionID))
			}
		}
	}

	return
}

// Plan is the top-level representation of the json format of a plan. It includes
// the complete config and current state.
type plan struct {
	FormatVersion    string `json:"format_version,omitempty"`
	TerraformVersion string `json:"terraform_version,omitempty"`
	//Variables        variables   `json:"variables,omitempty"`
	//PlannedValues    stateValues `json:"planned_values,omitempty"`
	// ResourceChanges are sorted in a user-friendly order that is undefined at
	// this time, but consistent.
	ResourceChanges []resourceChange  `json:"resource_changes,omitempty"`
	OutputChanges   map[string]change `json:"output_changes,omitempty"`
	PriorState      json.RawMessage   `json:"prior_state,omitempty"`
	Config          json.RawMessage   `json:"configuration,omitempty"`
}

// resourceChange is a description of an individual change action that Terraform
// plans to use to move from the prior state to a new state matching the
// configuration.
type resourceChange struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// ModuleAddress is the module portion of the above address. Omitted if the
	// instance is in the root module.
	ModuleAddress string `json:"module_address,omitempty"`

	// "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
	//Index        addrs.InstanceKey `json:"index,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`

	// "deposed", if set, indicates that this action applies to a "deposed"
	// object of the given instance rather than to its "current" object. Omitted
	// for changes to the current object.
	Deposed string `json:"deposed,omitempty"`

	// Change describes the change that will be made to this object
	Change change `json:"change,omitempty"`
}

// Change is the representation of a proposed change for an object.
type change struct {
	// Actions are the actions that will be taken on the object selected by the
	// properties below. Valid actions values are:
	//    ["no-op"]
	//    ["create"]
	//    ["read"]
	//    ["update"]
	//    ["delete", "create"]
	//    ["create", "delete"]
	//    ["delete"]
	// The two "replace" actions are represented in this way to allow callers to
	// e.g. just scan the list for "delete" to recognize all three situations
	// where the object will be deleted, allowing for any new deletion
	// combinations that might be added in future.
	Actions []string `json:"actions,omitempty"`

	// Before and After are representations of the object value both before and
	// after the action. For ["create"] and ["delete"] actions, either "before"
	// or "after" is unset (respectively). For ["no-op"], the before and after
	// values are identical. The "after" value will be incomplete if there are
	// values within it that won't be known until after apply.
	Before       json.RawMessage `json:"before,omitempty"`
	After        json.RawMessage `json:"after,omitempty"`
	AfterUnknown json.RawMessage `json:"after_unknown,omitempty"`
}
