package config

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// use a single instance of Validate, it caches struct info
var validate = validator.New()

// Config struct is a representation of the environment variables passed into the container
type Config struct {
	// Set by container overrides
	AccountID                 string `envconfig:"TERRASCALE_ACCOUNT_ID"` // The subscription id to deploy to
	TerrascaleTargetAccountID string `envconfig:"TERRASCALE_ACCOUNT_ID"` // The target account being deployed to using the delivery framework (use ACCOUNT_ID env for compatibility)
	// CredsID                         string   `envconfig:"TERRASCALE_CREDS_ID"`                                               // The identifier that determines which set of credentials to use (for which tenant)
	TerrascaleReleaseDeploymentID string `envconfig:"TERRASCALE_RELEASE_DEPLOYMENT_ID"` // The execution id of the CodePipeline that triggered these tasks
	TerrascaleRingDeploymentID    string `envconfig:"TERRASCALE_RING_DEPLOYMENT_ID"`    // The name of the Step Fn that triggered these tasks
	// UpdateStatusLambda              string   `envconfig:"UPDATE_STATUS_LAMBDA"`                                              // The name of the Lambda that is invoke to update the deployment status
	TerrascaleTargetRegions      []string `envconfig:"TERRASCALE_TARGET_REGIONS"`                                         // Terrascale will apply regional step deployments across these regions
	TerrascaleRegionGroup        string   `envconfig:"TERRASCALE_REGION_GROUP" validate:"eq=us|eq=eu|eq=uk" default:"us"` // The identified region group being executed in, this will derive primary region for primary step deployments; MUST NOT contain spaces, underscores or hypens
	TerrascaleRegionGroupRegions []string `envconfig:"TERRASCALE_REGION_GROUP_REGIONS"`                                   // Terrascale will execute regional step deployments across these regions, running destroy in the regions that do not intersect with `TERRASCALE_TARGET_REGIONS`
	UniqueExternalExecutionID    string
	CSP                          string   `envconfig:"TERRASCALE_CSP" required:"true" validate:"eq=AZU|eq=AWS|eq=GCP"` // CSP being run against (CloudServiceProvider)
	DeploymentRing               string   `envconfig:"TERRASCALE_DEPLOYMENT_RING"`
	SelfDestroy                  bool     `envconfig:"TERRASCALE_SELF_DESTROY"`              // Destroy will automatically execute Terraform Destroy after running deployments & tests
	DryRun                       bool     `envconfig:"TERRASCALE_DRY_RUN"`                   // DryRun will only execute up to Terraform plan, describing what will happen if deployed
	StepWhitelist                []string `envconfig:"TERRASCALE_STEP_WHITELIST"`            // Target_Steps is a comma separated list of step ids to reflect the whitelisted steps to be executed, e.g. core#logging#final_destination_bucket, core#logging#bridge_azu
	TargetAll                    bool     `envconfig:"TERRASCALE_TARGET_ALL" default:"true"` // This is a global whitelist and overrules targeted tracks and targeted steps, primarily for dev and testing
	// CommonRegion                    string   `envconfig:"TERRASCALE_COMMON_REGION" default:"us-east-1"`
	// AccountOwnerLabel               string   `envconfig:"TERRASCALE_ACCOUNT_OWNER"` // Owner's ID of the passed in ACCOUNT_ID
	Version                         string          `envconfig:"TERRASCALE_VERSION"` // Version override
	MaxRetries                      int             `envconfig:"TERRASCALE_MAX_RETRIES" default:"3"`
	MaxTestRetries                  int             `envconfig:"TERRASCALE_MAX_TEST_RETRIES" default:"2"`
	LogLevel                        string          `envconfig:"TERRASCALE_LOG_LEVEL" default:"info"`
	TerrascalePrimaryRegionOverride string          `envconfig:"TERRASCALE_PRIMARY_REGION"`
	CoreAccounts                    CoreAccountsMap `envconfig:"TERRASCALE_CORE_ACCOUNTS"`
	RegionGroups                    RegionGroupsMap `envconfig:"TERRASCALE_REGION_GROUPS"`
	// Set at task definition creation
	Namespace   string `envconfig:"TERRASCALE_NAMESPACE"`                   // The namespace to use in the Terraform run.
	Environment string `required:"true" envconfig:"TERRASCALE_ENVIRONMENT"` // The name of the environment (e.g. pr, nonprod, prod) which comes from the CodeBuild project
	Project     string `required:"true" envconfig:"TERRASCALE_PROJECT" default:"terrascale"`
}

type RegionGroupsMap map[string]map[string][]string

func (ipd *RegionGroupsMap) Decode(value string) error {
	return json.Unmarshal([]byte(value), ipd)
}

type CoreAccountsMap map[string]Account

func (ipd *CoreAccountsMap) Decode(value string) error {
	return json.Unmarshal([]byte(value), ipd)
}

// Deployment ...
type Deployment struct {
	Phase         string
	Result        string
	ResultMessage string
	Config        Config
	//DeployMetadata        DeployMetadata
}

// DeployMetadata ...
type DeployMetadata struct {
	Version   string `json:"version"`
	Region    string `json:"pr_region"`
	BaseImage string `json:"base_image"`
}

// Account is a struct that represents details about an Account
type Account struct {
	ID                string
	CredsID           string
	CSP               string
	AccountOwnerLabel string
}

// GetPrimaryRegionByCSP retrieves the primary region by CSP
func (cfg Config) GetPrimaryRegionByCSP(csp string) string {
	// support adhoc targeting of other primary regions, in e.g. ephemeral environments
	if cfg.TerrascalePrimaryRegionOverride != "" {
		return cfg.TerrascalePrimaryRegionOverride
	}

	if cfg.RegionGroups == nil {
		cfg.RegionGroups = GetDefaultRegionGroups()
	}

	return cfg.RegionGroups[strings.ToLower(csp)][strings.ToLower(cfg.TerrascaleRegionGroup)][0]
}

func GetDefaultRegionGroups() map[string]map[string][]string {
	return map[string]map[string][]string{
		"azu": {
			"us": []string{"centralus"},
			"uk": []string{"uksouth"},
			"eu": []string{"eu-north-1"},
		},
		"aws": {
			"us": []string{"us-east-1"},
			"uk": []string{"eu-west-2"},
			"eu": []string{"northeurope"},
		},
		"gcp": {
			"us": []string{"us-central1"},
			"uk": []string{"europe-west2"},
			"eu": []string{"europe-north1"},
		},
	}
}

// GetConfig retrieves a deployment config
func GetConfig() (config Config, err error) {
	validate.RegisterStructValidation(InputValidation, Config{})

	err = envconfig.Process("", &config)

	if err != nil {
		return
	}

	config.TerrascaleRegionGroup = strings.ToLower(config.TerrascaleRegionGroup)

	// if not set externally, set to legacy defaults
	if len(config.RegionGroups) == 0 {
		config.RegionGroups = GetDefaultRegionGroups()
	}

	// if not regions specifically targeted, default to primary region
	if len(config.TerrascaleTargetRegions) == 0 {
		config.TerrascaleTargetRegions = []string{config.GetPrimaryRegionByCSP(config.CSP)}
	}

	err = validate.Struct(config)

	return
}

func InputValidation(sl validator.StructLevel) {
	input := sl.Current().Interface().(Config)

	loweredEnv := strings.ToLower(input.Environment)
	if loweredEnv == "pr" || loweredEnv == "local" {
		// If running in pr or local, namespace is required. Except when executing a dryrun.
		// This prevents developers from modifying a higher environment from their local device
		if input.Namespace == "" && input.DryRun == false {
			sl.ReportError(input.Namespace, "namespace", "Namespace", "required-in-pr-local-when-dryrun-false", "")
		}
	}
}
