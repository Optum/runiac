# Terrascale

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
