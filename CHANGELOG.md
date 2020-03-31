# Terrascale

## 0.0.5 (Mar 28, 2020)

Expanded `FeatureToggleDisableCreds` feature toggle

## 0.0.4 (Mar 28, 2020)

Updated the `backend.tf` parser to interpolate the following fields consistently:
 
- `key`
- `bucket`
- `role_arn`

## 0.0.3 (Mar 27, 2020)

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