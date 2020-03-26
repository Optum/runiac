## Gaia Deploy Container 0.15.1 (Mar 25, 2020)

##### BUG FIXES

- Fixes defect that was preventing gaia.yaml from functioning properly at the track level.

## Gaia Deploy Container 0.15.0 (Mar 24, 2020)

##### ENHANCEMENTS

Continuing to remove the "hidden magic" areas with 3 more input configurations relying on more definition within terraform:

- `GAIA_FEATURE_DISABLE_CREDS`: Disables the "auto pulling" of creds based on accts CREDS_ID.  This would be true if you'd like to use creds passed into container
- `GAIA_FEATURE_DISABLE_S3_BACKEND_DEFAULT_BUCKET`: Disables setting the backend bucket, utilizing what is set in backend tf file.
- `GAIA_FEATURE_DISABLE_S3_BACKEND_KEY_PREFIX`: Disables setting a standardized account key prefix

## Gaia Deploy Container 0.14.1 (Feb 12, 2020)

##### REFACTOR

- Two new logging fields to support launchpad logging
  - `lpclagg = ${gaiaRingDeploymentID}`
  - `lpcltype = gaia`

## Gaia Deploy Container 0.14.0 (Feb 12, 2020)

##### ENHANCEMENTS

- Updates to the secure cli to support creating multiple groups and passing through payload

## Gaia Deploy Container 0.13.0 (Feb 28, 2020)

##### ENHANCEMENTS

- Start passing through raw aws credentials to terraform (allowing terraform to handle assume roles)
  - To avoid breaking changes this **only** occurs when `CSP == azure` and `backend == s3` with `role_arn` set.   

## Gaia Deploy Container 0.12.0 (Feb 26, 2020)

##### ENHANCEMENTS

- Support `${var.core_account_ids_map}` variable in `backend.tf` file.

## Gaia Deploy Container 0.11.0 (Feb 25, 2020)

##### ENHANCEMENTS

- Added `execute_when.region_in` to step gaia configuration yaml. See [README.md](./README.md) for more info.

## Gaia Deploy Container 0.10.5 (Feb 25, 2020)

##### ENHANCEMENTS

- Upgraded Terraform to v0.12.21

## Gaia Deploy Container 0.10.4 (Feb 24, 2020)

##### ENHANCEMENTS

- Add supported for two new Azure Tenants
  - UHGUK
  - DSISTG

## Gaia Deploy Container 0.10.3 (Feb 21, 2020)

##### BUG FIXES

- Changed evaluation criteria in stepexecution.go L:86 to `err == nil` from `err != nil` to fix AZU rotater creds in bridge stream issue

## Gaia Deploy Container 0.10.1 (Feb 10, 2020)

##### BUG FIXES

- Fixes incorrect `region` logging field value for certain primary region deployments

## Gaia Deploy Container 0.10.1 (Feb 10, 2020)

##### BUG FIXES

- Fixes AZU credential retrieval failing for required `credsid` parameter.

## Gaia Deploy Container 0.10.0 (Feb 10, 2020)

##### ENHANCEMENTS

- Adds ability to configure track and step execution configuration via a `gaia.yaml` file

## Gaia Deploy Container 0.9.4 (Feb 6, 2020)

##### ENHANCEMENTS

- Updates to CSP credential handling. In the case of AZU Launchpad, the bridge account steps will now have the
  credentials for the environments bridge account and the target AZU account.

## Gaia Deploy Container 0.9.3 (Jan 30, 2020)

##### BREAKING CHANGES

- Core Accounts and Region Groups now **must** be configured via environment variables

## Gaia Deploy Container 0.9.2 (Jan 30, 2020)

##### ENHANCEMENTS

- Added support for passing in core account definition as a json string environment variable, `GAIA_CORE_ACCOUNTS`
- Added support for passing in region groups definition as a json string environment variable, `GAIA_REGION_GROUPS`

## Gaia Deploy Container 0.9.1 (Jan 22, 2020)

##### ENHANCEMENTS

- Added support for scheduled Gaia runs to CUSTOMER stage, initially targeting `INTERNAL` ring only

## Gaia Deploy Container 0.9.0 (Dec 13, 2019)

Major theme of these changes is to modularize gaia to be executed in other CI environments and decouple from launchpad specific use cases.

##### BREAKING CHANGES

- `NAMESPACE` will now be passed to all environments, not just `local` and `pr`
  - NOTE: this also includes statefiles, ensure higher environments **do not** pass in `NAMESPACE` for backwards compatibility
- Dynamodb deployment tracking is now disabled by default.
  - To enable this feature, set the `GAIA_REPORTER_DYNAMODB` configuration value to `true`

##### ENHANCEMENTS

- Added `LOG_FORMAT` field supports a `gaia` value to output human readable, colored code executions
- Added logging fields `environment` and `namespace`

## Gaia Deploy Container 0.8.25 (Dec 13, 2019)

##### REFACTOR

- Modified ending log output format

## Gaia Deploy Container 0.8.24 (Dec 13, 2019)

##### ENHANCEMENTS

- Pull Requests now support multi-region deployments (two regions per PR)
  - The `version.json` file that is updated by the Launchpad Jenkins pipelines now supports a comma separated string for the `pr_region` field. The first value in this list of regions becomes the primary region and the string itself is used to construct the target regions. For example, if `pr_region` is set to `"centralus,westus"`, `centralus` becomes the primary region during the PR deployment and `centralus` and `westus` become the target regions during the PR deployment.

##### BUG FIXES

- Fixed Lambda execution errors on tests

## Gaia Deploy Container 0.8.23 (Nov 21, 2019)

##### ENHANCEMENTS

- Fetch params from AWS only once

## Gaia Deploy Container 0.8.22 (Nov 21, 2019)

##### REFACTOR

- Removed legacy code

## Gaia Deploy Container 0.8.21 (Nov 21, 2019)

##### ENHANCEMENTS

- Added new variable `gaia_region_group_regions` to steps for the list of regions in a particular Region Group

## Gaia Deploy Container 0.8.20 (Nov 18, 2019)

##### ENHANCEMENTS

- Added support for terraform overrides via a `override` directory in step. See more information in the README [here](./README.md#override-files).

## Gaia Deploy Container 0.8.19 (Nov 18, 2019)

##### REFACTOR

- First iteration creating an easier to follow step package in a new `Stepper` interface struct: `TerraformStepper2`

## Gaia Deploy Container 0.8.18 (Nov 14, 2019)

##### ENHANCEMENTS

- Added AWS Organizations master accounts to mapping of credentials available to Gaia steps

## Gaia Deploy Container 0.8.17 (Nov 14, 2019)

##### ENHANCEMENTS

- Step tests now receive all the same input variables as the step's themselves

## Gaia Deploy Container 0.8.16 (Nov 12, 2019)

##### ENHANCEMENTS

- Added `var.gaia_region_group` for all steps
- Added `var.gaia_primary_region` for all steps

## Gaia Deploy Container 0.8.15 (Nov 8, 2019)

##### ENHANCEMENTS

- Added `var.gaia_target_account_id` for all steps
- Updated to terraform `v0.12.12`

##### BUGFIXES

- Added back `retryCount` logging field

## Gaia Deploy Container 0.8.14 (Nov 6, 2019)

##### ENHANCEMENTS

- Added caching for Azure credentials

## Gaia Deploy Container 0.8.13 (Oct 30, 2019)

##### ENHANCEMENTS

- Allow empty `NAMESPACE` when executing in `ENVIRONMENT="PR|LOCAL" && DRYRUN=false`
  - This allows local execution to dry run higher environments

## Gaia Deploy Container 0.8.12 (Oct 29, 2019)

##### BUG FIXES

- Resolved multi-region shared state concurrency issue where regions regional output parameter values were overwritten from another region

## Gaia Deploy Container 0.8.11 (Oct 28, 2019)

##### BUG FIXES

- Resolved `nil` reference error when using `Local` terraform backend

## Gaia Deploy Container 0.8.10 (Oct 25, 2019)

##### ENHANCEMENTS

- Added support for `regional` variables while executing with `DESTROY` and `AUTO_DESTROY` enabled

## Gaia Deploy Container 0.8.9 (Oct 22, 2019)

##### ENHANCEMENTS

- Added support for Azure Core deployments. These are supported through setting the `subscription_id` field in the `azurerm` provider. See more information in the README [here](<./README.md#Provider-(Azurerm)>).
- Added `tenant_core_azu` to the core accounts mapping for Azure deployments. This represents the central subscription in each tenant.
- Added `gaia_deploy` to the core accounts mapping. This represents the AWS account that the deployment is running out of (the account in which the Fargate tasks are executing) and the account which SSM parameters are retrieved from.
- Added `pr_gaia_deploy` to the core accounts mapping. This represents the AWS Gaia account that PR's run out of. This allows prod deployments to add secret values to SSM parameter store in the PR Gaia account. A use case for this is generating credentials in `launchpad_core_azu` for the tenant that PR's run in and then using those shared credentials for Local/PR development since Local/PR development can only access the PR Gaia account.

## Gaia Deploy Container 0.8.8 (Oct 18, 2019)

##### BUG FIXES

- Removed `github.optum.com` references in `go.mod` and `go.sum` - unable to pull these dependencies in Codebuild.

## Gaia Deploy Container 0.8.7 (Oct 18, 2019)

##### ENHANCEMENTS

- Allows `${var.gaia_deployment_ring}` variable to be used in `S3 backend.key` configuration

## Gaia Deploy Container 0.8.6 (Oct 16, 2019)

##### ENHANCEMENTS

- Allows `LOCAL` deployment ring

## Gaia Deploy Container 0.8.5 (Oct 15, 2019)

##### BUG FIXES

- Fixed a concurrency issue when executing multiple tracks

## Gaia Deploy Container 0.8.4 (Oct 11, 2019)

##### BUG FIXES

- Fixed an issue where all parameters from `SSM` were not being retrieved

## Gaia Deploy Container 0.8.3 (Oct 9, 2019)

##### BUG FIXES

- Fixed FlushTrack inconsistencies

## Gaia Deploy Container 0.8.2 (Oct 8, 2019)

##### BUG FIXES

- Fixed destroy not being executed

## Gaia Deploy Container 0.8.1 (Oct 4, 2019)

##### ENHANCEMENTS

- `terraform show` and `terraform output` no longer stream to stdout. This prevents outputs that are marked as sensitive from displaying in the console.

## Gaia Deploy Container 0.8.0 (Oct 1, 2019)

Major release

##### ENHANCEMENTS

- Added support for regional deployments, see [README](https://github.optum.com/CommercialCloud-Team/Gaia_common/tree/master/customer/base_deploy_image#step_deployment_types) for further details.
  - Additional configuration variables `GAIA_TARGET_REGIONS` and `GAIA_REGION_GROUP`
- Added common deploy role `GaiaDeploy`. This also helps to provide a constant role for local usage.
- Added default input variables `gaia_stage`, `gaia_track`, `gaia_step`, `gaia_region_deploy_type` and `region`.

##### BREAKING CHANGES

- Removed default input variable `aws_region` in favor of `region`.

## Gaia Deploy Container 0.7.1 (Sept 20, 2019)

##### ENHANCEMENTS

- Added logging fields: `accountOwnerMSID`, `gaiaRingDeploymentID` and `gaiaReleaseDeploymentID`

## Gaia Deploy Container 0.7.0 (Sept 16, 2019)

##### ENHANCEMENTS

- Steps in the same track will receive the previous step's output variables as input environment variables. See [README.md](https://github.optum.com/CommercialCloud-Team/Gaia_common/tree/master/customer/base_deploy_image#using_previous_step_output_variables)

## Gaia Deploy Container 0.6.3 (Sept 11, 2019)

##### ENHANCEMENTS

- Added support for Azure

## Gaia Deploy Container 0.6.2 (Sept 11, 2019)

##### ENHANCEMENTS

- Added `gaia_deployment_ring` as default environment input variable for terraform.

## Gaia Deploy Container 0.6.1 (Sept 7, 2019)

##### ENHANCEMENTS

- Tests will now retry two times with a 20 second delay.

## Gaia Deploy Container 0.6.0 (Sept 7, 2019)

##### ENHANCEMENTS

- Added tfstate directory support for `gaiaTargetAccountID` (see [README.md](https://github.optum.com/CommercialCloud-Team/Gaia_common/tree/master/customer/base_deploy_image#key))

## Gaia Deploy Container 0.5.4 (Sept 7, 2019)

##### ENHANCEMENTS

- Added `gaiaTargetAccountID` to logging fields
- Terraform version changed to `0.12.8`

## Gaia Deploy Container 0.5.3 (August 30, 2019)

##### BUG FIXES

- Fixed incorrect reporting on the number of tracks being executed

## Gaia Deploy Container 0.5.2 (August 29, 2019)

##### ENHANCEMENTS

- Terraform version changed to `0.12.7`

## Gaia Deploy Container 0.5.1 (August 29, 2019)

##### ENHANCEMENTS

- Added `TF_VAR_gaia_target_account_id` as an environment variable that is exported for Terraform commands
  - This value is the same as `ACCOUNT_ID` currently. The plan is to replace `ACCOUNT_ID` with `TF_VAR_gaia_target_account_id`. This value represents the account id that the step function told the fargate task to deploy to

## Gaia Deploy Container 0.5.0 (August 28, 2019)

##### ENHANCEMENTS

- Added `BR_STEP_WHITELIST` configuration option to whitelist which steps get executed, ie. `#core#logging#final_destination_bucket`
- Added `BR_TARGET_ALL` to override `BR_STEP_WHITELIST` and whitelist all steps, defaults to false.

##### BREAKING CHANGES

- Removed `BR_TRACKS` configuration option
- `BR_TARGET_ALL` now defaults to false. To execute steps either `BR_TARGET_ALL` is true or `BR_STEP_WHITELIST` is set.

## Gaia Deploy Container 0.4.5 (August 23, 2019)

##### REFACTOR

- Modify local logs to output json

## Gaia Deploy Container 0.4.4 (August 21, 2019)

##### BUG FIXES

- Moved retry logic to include `apply` and `plan` to resolve issues where first `apply` would partially complete and retries encountered `Error: plan is stale`.

## Gaia Deploy Container 0.4.3 (August 20, 2019)

##### BUG FIXES

- Resolved issue where backend config bucket was not set correctly in `ByCount` execution model (launchpad_core_aws)

## Gaia Deploy Container 0.4.2 (August 20, 2019)

##### REFACTOR

- Log `stderr` as `error` level logs
- Better consistency between step `test` and `deploy`

## Gaia Deploy Container 0.4.1 (August 16, 2019)

##### BUG FIXES

- Fixes multiple bugs when defining a custom `backend.key` value with a path would be parsed into an incorrect end result.

## Gaia Deploy Container 0.4.0 (August 9, 2019)

##### ENHANCEMENTS

- Adds support for using parameter store as the credential store for delivery framework (replacing Jenkins Credentials). See [Readme](/README.md) for more information.

## Gaia Deploy Container 0.3.1 (August 13, 2019)

##### BUG FIXES

- Increased buffer size when reading from stdout/stderr. Builds would fail for runs with a large number of changes (e.g. the Gaia VPC) when running `terraform show`.

## Gaia Deploy Container 0.3.0 (August 9, 2019)

##### ENHANCEMENTS

- Added retries to any Terraform commands that are run from the `terraform` package.

## Gaia Deploy Container 0.2.1 (Aug 8, 2019)

##### BUG FIXES

- Fixes "No such file found" error for steps that do not have a provider.tf file.

## Gaia Deploy Container 0.2.0 (July 29, 2019)

##### ENHANCEMENTS

- Providing support for `provider.assume_role.role_arn`, explicity allowing steps to define which accounts they are deploying to. Meaning a container execution could end up deploying to multiple accounts.
- Updates to execute update status for each executed step.
- Updates to retrieve credentials for each executed step.

This is being used to handle the Launchpad Core Accounts.

##### BREAKING CHANGES

- Adds a default variable of `core_account_ids_map` to all deploys

```hcl-terraform
variable "core_account_ids_map" {
  type        = map(string)
  description = "Mapping of the available core account ids, AWS: logging_final_destination, guard_duty_master, logging_bridge_aws, logging_bridge_azu"
}
```
