# Terrascale

## 0.0.3 (Mar 27, 2020)

Updated the `backend.tf` parser to interpolate `var.gaia_step` for the key attribute.

## 0.0.2 (Mar 26, 2020)

Back-porting a few missed commits

### ENHANCEMENTS

- Updated the `backend.tf` parser to interpolate `var.gaia_target_account_id` for the role_arn attribute.
- Updated the `providers.tf` parser to interpolate `var.gaia_target_account_id` for the provider attributes.

## 0.0.1 (Mar 26, 2020)

Initial Release