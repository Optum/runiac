package config

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.optum.com/healthcarecloud/terrascale/pkg/params"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.optum.com/healthcarecloud/terrascale/pkg/auth"

	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var FargateTaskMetadataEndpoint = "http://169.254.170.2/v2/metadata"
var httpClient = &http.Client{Timeout: 10 * time.Second}

// use a single instance of Validate, it caches struct info
var validate = validator.New()

// Config struct is a representation of the environment variables passed into the container
type Config struct {
	// Set by container overrides
	AccountID                                   string   `envconfig:"ACCOUNT_ID"`                                                  // The subscription id to deploy to
	GaiaTargetAccountID                         string   `envconfig:"ACCOUNT_ID"`                                                  // The target account being deployed to using the delivery framework (use ACCOUNT_ID env for compatibility)
	CredsID                                     string   `envconfig:"CREDS_ID"`                                                    // The identifier that determines which set of credentials to use (for which tenant)
	GaiaReleaseDeploymentID                     string   `envconfig:"CODEPIPELINE_EXECUTION_ID"`                                   // The execution id of the CodePipeline that triggered these tasks
	GaiaRingDeploymentID                        string   `envconfig:"TERRASCALE_RING_DEPLOYMENT_ID"`                                     // The name of the Step Fn that triggered these tasks
	UpdateStatusLambda                          string   `envconfig:"UPDATE_STATUS_LAMBDA"`                                        // The name of the Lambda that is invoke to update the deployment status
	GaiaTargetRegions                           []string `envconfig:"TERRASCALE_TARGET_REGIONS"`                                         // Gaia will apply regional step deployments across these regions
	GaiaRegionGroup                             string   `envconfig:"TERRASCALE_REGION_GROUP" validate:"eq=us|eq=eu|eq=uk" default:"us"` // The identified region group being executed in, this will derive primary region for primary step deployments; MUST NOT contain spaces, underscores or hypens
	GaiaRegionGroupRegions                      []string `envconfig:"TERRASCALE_REGION_GROUP_REGIONS"`                                   // Gaia will execute regional step deployments across these regions, running destroy in the regions that do not intersect with `TERRASCALE_TARGET_REGIONS`
	FargateTaskID                               string
	CSP                                         string   `required:"true" validate:"eq=AZU|eq=AWS|eq=GCP"` // CSP being run against (CloudServiceProvider)
	DeploymentRing                              string   `envconfig:"DEPLOYMENT_RING"`
	SelfDestroy                                 bool     `envconfig:"TERRASCALE_SELF_DESTROY"`   // Destroy will automatically execute Terraform Destroy after running deployments & tests
	DryRun                                      bool     `envconfig:"TERRASCALE_DRY_RUN"`        // DryRun will only execute up to Terraform plan, describing what will happen if deployed
	StepWhitelist                               []string `envconfig:"TERRASCALE_STEP_WHITELIST"` // Target_Steps is a comma separated list of step ids to reflect the whitelisted steps to be executed, e.g. core#logging#final_destination_bucket, core#logging#bridge_azu
	TargetAll                                   bool     `envconfig:"TERRASCALE_TARGET_ALL"`     // This is a global whitelist and overrules targeted tracks and targeted steps, primarily for dev and testing
	CommonRegion                                string   `envconfig:"TERRASCALE_COMMON_REGION" default:"us-east-1"`
	AccountOwnerMSID                            string   `envconfig:"ACCOUNT_OWNER"` // Owner's MSID of the passed in ACCOUNT_ID
	Version                                     string
	LogLevel                                    string `envconfig:"LOG_LEVEL" default:"info"`
	GaiaPrimaryRegionOverride                   string
	CoreAccounts                                CoreAccountsMap `envconfig:"TERRASCALE_CORE_ACCOUNTS"`
	RegionGroups                                RegionGroupsMap `envconfig:"TERRASCALE_REGION_GROUPS"`
	FeatureToggleDisableCreds                   bool            `envconfig:"TERRASCALE_FEATURE_DISABLE_CREDS"`                      // Disables the "auto pulling" of creds based on accts CREDS_ID.  This would be true if you'd like to use creds passed into container
	FeatureToggleDisableBackendDefaultBucket    bool            `envconfig:"TERRASCALE_FEATURE_DISABLE_S3_BACKEND_DEFAULT_BUCKET"`  // Disables setting the backend bucket, utilizing what is set in backend tf file.
	FeatureToggleDisableS3BackendKeyPrefix      bool            `envconfig:"TERRASCALE_FEATURE_DISABLE_S3_BACKEND_KEY_PREFIX"`      // Disables setting a standardized account key prefix
	FeatureToggleDisableS3BackendKeyNamespacing bool            `envconfig:"TERRASCALE_FEATURE_DISABLE_S3_BACKEND_KEY_NAMESPACING"` // Disables the usage of namespace, region, and region deploy type to automatically create state file
	FeatureToggleDisableParamStoreVars          bool            `envconfig:"FEATURE_TOGGLE_DISABLE_PARAM_STORE_VARS"`
	// Set at task definition creation
	Namespace        string `required:"true" envconfig:"NAMESPACE"`                                   // The namespace to use in the Terraform run. This should only be used when ENVIRONMENT != prod
	Environment      string `required:"true" validate:"eq=prod|eq=pr|eq=nonprod|eq=local|eq=jenkins"` // The name of the environment (e.g. pr, nonprod, prod) which comes from the CodeBuild project
	ReporterDynamodb bool   `envconfig:"TERRASCALE_REPORTER_DYNAMODB"`
	Authenticator    auth.Authenticator
	StepParameters   params.StepParameters
	Stage            string `envconfig:"TERRASCALE_STAGE"`
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
	Phase                 string
	Result                string
	ResultMessage         string
	Config                Config
	DeployMetadata        DeployMetadata
	PlatformAccessSession *session.Session
}

// DeployMetadata ...
type DeployMetadata struct {
	Version   string `json:"version"`
	Region    string `json:"pr_region"`
	BaseImage string `json:"base_image"`
}

// Account is a struct that represents details about an Account
type Account struct {
	ID               string
	CredsID          string
	CSP              string
	AccountOwnerMSID string
}

// GetPrimaryRegionByCSP retrieves the primary region by CSP
func (cfg Config) GetPrimaryRegionByCSP(csp string) string {
	// support adhoc targeting of other primary regions, ie pull requests and local environments
	if strings.ToLower(csp) == strings.ToLower(cfg.CSP) && cfg.GaiaPrimaryRegionOverride != "" {
		return cfg.GaiaPrimaryRegionOverride
	}

	if cfg.RegionGroups == nil {
		cfg.RegionGroups = GetDefaultRegionGroups()
	}

	return cfg.RegionGroups[strings.ToLower(csp)][strings.ToLower(cfg.GaiaRegionGroup)][0]
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

	config.GaiaRegionGroup = strings.ToLower(config.GaiaRegionGroup)

	// if not set externally, set to legacy defaults
	if len(config.RegionGroups) == 0 {
		config.RegionGroups = GetDefaultRegionGroups()
	}

	// if not regions specifically targeted, default to primary region
	if len(config.GaiaTargetRegions) == 0 {
		config.GaiaTargetRegions = []string{config.GetPrimaryRegionByCSP(config.CSP)}
	}

	// backwards compatibility
	if os.Getenv("TERRASCALE_SELF_DESTROY") == "" && os.Getenv("BR_AUTO_DESTROY") == "true" {
		config.SelfDestroy = true
	}

	if config.GaiaRingDeploymentID == "" {
		config.GaiaRingDeploymentID = os.Getenv("STEP_FUNCTION_NAME")
	}

	err = validate.Struct(config)

	return
}

func GetVersionJSON(log *logrus.Entry, fs afero.Fs, file string) (versionJSON DeployMetadata, err error) {
	// Open our jsonFile
	jsonFile, err := fs.Open(file)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Error(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened xmlFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal(byteValue, &versionJSON)
	if versionJSON.Region == "" {
		versionJSON.Region = "us-east-1"
	}

	return
}

type FargateTaskMetadata struct {
	TaskARN string
}

func GetRunningFargateTaskID(environment string) (string, error) {
	if environment == "local" || environment == "jenkins" {
		u, _ := uuid.NewV4()
		return u.String(), nil
	}

	req, err := http.NewRequest("GET", FargateTaskMetadataEndpoint, nil)

	req.Header.Add("cache-control", "no-cache")

	taskMetadata := &FargateTaskMetadata{}
	err = getJson(FargateTaskMetadataEndpoint, taskMetadata)
	if err != nil {
		return "", err
	}
	return strings.Split(taskMetadata.TaskARN, ":task/")[1], err
}

func getJson(url string, target interface{}) error {
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
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

	if input.ReporterDynamodb {
		if input.GaiaRingDeploymentID == "" {
			sl.ReportError(input.GaiaRingDeploymentID, "TERRASCALE_RING_DEPLOYMENT_ID", "GaiaRingDeploymentID", "required-with-reporter-dynamodb", "")
		}
		if input.UpdateStatusLambda == "" {
			sl.ReportError(input.UpdateStatusLambda, "UPDATE_STATUS_LAMBDA", "UpdateStatusLambda", "required-with-reporter-dynamodb", "")
		}
	}
}
