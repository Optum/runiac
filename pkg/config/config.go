package config

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// use a single instance of Validate, it caches struct info
var validate = validator.New()

// Config struct is a representation of the environment variables passed into the container
type Config struct {
	// Set by container overrides
	AccountID       string   `mapstructure:"account_id"`       // The cloud account id to deploy to (AWS Account, Azure Subscription or GCP Project)
	TargetAccountID string   `mapstructure:"account_id"`       // The target account being deployed to using the delivery framework (use ACCOUNT_ID env for compatibility)
	RegionalRegions []string `mapstructure:"regional_regions"` // runiac will apply regional step deployments across these regions
	PrimaryRegion   string   `mapstructure:"primary_region" required:"true"`
	DryRun          bool     `mapstructure:"dry_run"` // DryRun will only execute up to Terraform plan, describing what will happen if deployed

	UniqueExternalExecutionID string
	DeploymentRing            string `mapstructure:"deployment_ring"`
	SelfDestroy               bool   `mapstructure:"self_destroy"` // Destroy will automatically execute Terraform Destroy after running deployments & tests
	RegionGroup               string
	StepWhitelist             []string        `mapstructure:"step_whitelist"` // Target_Steps is a comma separated list of step ids to reflect the whitelisted steps to be executed, e.g. core#logging#final_destination_bucket, core#logging#bridge_azu
	TargetAll                 bool            // This is a global whitelist and overrules targeted tracks and targeted steps, primarily for dev and testing
	Version                   string          `mapstructure:"version"` // Version override
	MaxRetries                int             `mapstructure:"max_retries"`
	MaxTestRetries            int             `mapstructure:"max_test_retries"`
	LogLevel                  string          `mapstructure:"log_level"`
	CoreAccounts              CoreAccountsMap `mapstructure:"core_accounts"`
	RegionGroups              RegionGroupsMap `mapstructure:"region_grouprs"`
	// Set at task definition creation
	Namespace   string `mapstructure:"namespace"`                   // The namespace to use in the Terraform run.
	Environment string `mapstructure:"environment" required:"true"` // The name of the environment (e.g. pr, nonprod, prod)
	Project     string `mapstructure:"project" required:"true"`
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

//func GetDefaultRegionGroups() map[string]map[string][]string {
//	return map[string]map[string][]string{
//		"azu": {
//			"us": []string{"centralus"},
//			"uk": []string{"uksouth"},
//			"eu": []string{"eu-north-1"},
//		},
//		"aws": {
//			"us": []string{"us-east-1"},
//			"uk": []string{"eu-west-2"},
//			"eu": []string{"northeurope"},
//		},
//		"gcp": {
//			"us": []string{"us-central1"},
//			"uk": []string{"europe-west2"},
//			"eu": []string{"europe-north1"},
//		},
//	}
//}

// GetConfig retrieves a deployment config
func GetConfig() (Config, error) {
	viper.SetConfigName("runiac") // name of config file (without extension)

	viper.AddConfigPath(".")
	viper.SetEnvPrefix("runiac")
	viper.AutomaticEnv()

	// https://github.com/spf13/viper/issues/188#issuecomment-255519149
	_ = viper.BindEnv("environment")
	_ = viper.BindEnv("namespace")
	_ = viper.BindEnv("project")
	_ = viper.BindEnv("log_level")
	_ = viper.BindEnv("dry_run")
	_ = viper.BindEnv("self_destroy")
	_ = viper.BindEnv("deployment_ring")
	_ = viper.BindEnv("primary_regions")
	_ = viper.BindEnv("regional_regions")
	_ = viper.BindEnv("max_retries")
	_ = viper.BindEnv("max_test_retries")
	_ = viper.BindEnv("account_id")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
		} else {
			return Config{}, err
		}
	}

	conf := &Config{
		MaxTestRetries: 2,
		MaxRetries:     3,
		LogLevel:       logrus.InfoLevel.String(),
		Project:        "runiac",
		TargetAll:      true,
	}
	err := viper.Unmarshal(conf)

	if err != nil {
		fmt.Printf("unable to decode into config struct, %v", err)
		return Config{}, err
	}

	validate.RegisterStructValidation(InputValidation, conf)

	err = validate.Struct(conf)

	if err != nil {
		return *conf, err
	}

	// if step whitelist is set, respect it
	if conf.TargetAll && len(conf.StepWhitelist) > 0 {
		conf.TargetAll = false
	}

	return *conf, nil
}

func InputValidation(sl validator.StructLevel) {
	input := sl.Current().Interface().(Config)

	if input.Environment == "" {
		sl.ReportError(input.Namespace, "environment", "environment", "required-environment", "")
	}

	if input.PrimaryRegion == "" {
		sl.ReportError(input.Namespace, "primary_region", "primaryRegion", "required-primary-region", "")
	}
}
