# Terrascale

## 0.3.1 (October 29, 2020)

### ENHANCEMENTS

- Upgrade Golang builder from 1.12.12 to 1.14 and point to Docker Hub-optimized UHG Artifactory endpoint
- Upgrade `gotestsum` to latest v0.5.2

## 0.3.0 (October 21, 2020)

### ENHANCEMENTS

- Updated Terraform to `0.13.4`. Please see the documentation [here](https://www.terraform.io/upgrade-guides/0-13.html) for upgrading to Terraform `v0.13`

### BREAKING CHANGES

- Projects upgrading to this version of Terrascale will need to update their filesystem layout for local copies of providers as stated [here](https://www.terraform.io/upgrade-guides/0-13.html#new-filesystem-layout-for-local-copies-of-providers)
- Projects are now fully responsible for creating the directories they will be use for plugin caching and their own [.terraformrc](https://www.terraform.io/docs/commands/cli-config.html) file

## 0.2.3 (October 9, 2020)

### ENHANCEMENTS

- Added the default region groups for GCP to the `GetDefaultRegionGroups` function and added the missing `EU` region groups to AWS and Azure

### BUG FIX

- Fixed the backend key interpolation for the core account ids map. Previously this was not interpolating the key correctly when there were references to multiple core accounts, e.g. `bootstrap-launchpad-${var.core_account_ids_map.logging_bridge_gcp}/${var.core_account_ids_map.gcp_core_project}/${var.gaia_deployment_ring}.tfstate`

## 0.2.2 (July 27, 2020)

### ENHANCEMENTS

- Additional support for override files for deployments.
  - `override.tf` files will be supported to all deployments, including Self-Destroy.
  - `destroy_override.tf` files will be supported to Self-Destroy deployments.
  - `destroy_ring_*ring-name*_override.tf` files will be added to Self-Destroy deployments for the specified deployment ring.

## 0.2.1 (July 21, 2020)

### BUG FIX

- Include `gaia.yml` configuration check for `execute_when.region_in` for destroy step executions, filtered steps previously were still being attempted to be destroyed.

## 0.2.0 (July 13, 2020)

### ENHANCEMENTS

- Added the option to execute a pre-track before all other tracks

## 0.1.3 (July 7, 2020)

### ENHANCEMENTS

- Add `PULL_REQUEST_TEMPLATE.md` file.

### BUG FIX

- Failed destroy steps will now be considered as an overall failure.
- Fix link in README for local instructions.

## 0.1.2 (June 18, 2020)

### BUG FIX

- Security fix: ssm parameter safety

## 0.1.1 (June 10, 2020)

### REFACTOR

- Default JSON Log output will now log `time` fields to nanosecond granularity
- Skipped steps from `gaia.yml` will be marked as `Na` instead of `Skipped`

## 0.1.0 (May 11, 2020)

### BREAKING CHANGES

- When an execution contains a failed step the exit code will now be `1`

### BUG FIXES

- Fix when using "just in time" tfvars from param store

## 0.0.9 (Apr 20, 2020)

### ENHANCEMENTS

- Added `FEATURE_TOGGLE_DISABLE_PARAM_STORE_VARS` environment configuration for users that do not need param store variable injection

## 0.0.8 (Apr 17, 2020)

### ENHANCEMENTS

- Added support for the `OPTUM TELEHEALTH` tenant

## 0.0.7 (Apr 8, 2020)

### BUG FIX

- Resolve summary inaccurately including steps skipped for not existing

## 0.0.6 (Apr 3, 2020)

### REFACTOR

- Improved logging when steps fail

## 0.0.5 (Mar 28, 2020)

### REFACTOR

Expanded `FeatureToggleDisableCreds` feature toggle

## 0.0.4 (Mar 28, 2020)

### ENHANCEMENTS

Updated the `backend.tf` parser to interpolate the following fields consistently:

- `key`
- `bucket`
- `role_arn`

## 0.0.3 (Mar 27, 2020)

### ENHANCEMENTS

Updated the `backend.tf` parser to interpolate the following variables for the s3 `key` attribute:

- `var.gaia_step`
- `var.gaia_region_deploy_type`
- `var.region`
- `local.namespace-` (temporary backwards compatibility variable)

## 0.0.2 (Mar 26, 2020)

Back-porting a few missed commits

### ENHANCEMENTS

- Updated the `backend.tf` parser to interpolate `var.gaia_target_account_id` for the role_arn attribute.
- Updated the `providers.tf` parser to interpolate `var.gaia_target_account_id` for the provider attributes.

## 0.0.1 (Mar 26, 2020)

Initial Release
