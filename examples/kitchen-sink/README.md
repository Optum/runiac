<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Advanced runiac Example](#advanced-runiac-example)
  - [Requirements](#requirements)
  - [Running](#running)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Advanced runiac Example

This example will provide a more complex deployment scenario involving multiple cloud providers and services.
* A pre-track that deploys PagerDuty monitoring resources
* A simple track that deploys a Docker image and a storage bucket to a Google Cloud Platform project
* A complex track that:
  * Primary deployment that deploys a shared Key Vault into a central Azure region
  * Regional deployments that deploy a Docker image into App Service instances in two distinct regions

The following steps assume you are running on a Linux or macOS system, but the process will most likely be similar on Windows.

## Requirements

See the top-level README for information on obtaining these items:

- An Azure subscription
- A Google Cloud Platform project
- A PagerDuty account

## Running

If you did not clone this repository, you can have runiac scaffold a copy of this example by running:

```bash
runiac new --url github.com/optum/runiac//examples/kitchen-sink
```

Assuming you've set up your subscription and login credentials, initialize a runiac project by running the following command in this
directory:

```bash
runiac init
```

Since this example relies on an Azure CLI and Google Cloud SDK installation, we request to use the image `optumopensource/runiac:v0.0.1-beta3-alpine-azure-gcloud` via the `--container` flag.
Note that `entrypoint.sh` is set up to prompt the user for when credentials are missing or expired. This facilitates getting the example working,
but in a real-world scenario, you'll most likely want to follow your Terraform providers' best practices for propagating credentials instead.

You can now execute runiac using the following runiac CLI command, replacing placeholders with your actual access details:

```bash
TF_VAR_pagerduty_token="your-pagerduty-token" \
TF_VAR_gcp_project_id="your-gcp-project-od" \
runiac deploy \
  -e nonprod \
  -a your-azure-subscription-id \
  --interactive \
  --container optumopensource/runiac:v0.0.1-beta3-alpine-azure-gcloud
```

This will deploy several tracks which target PagerDuty, Azure and GCP. Notice several things about this command:
* You can provide Terraform variables by providing `TF_VAR_*` environment variables.
* Specify `-e nonprod` to indicate we are deploying to a non-production environment.
* Specify `-a` to pass in the account ID of a cloud provider. Since we use both Azure and GCP, we'll provide the Azure subscription ID here.
* Specify `--interactive` to run the runiac container in interactive mode (we need this to interact with the various CLIs).
  * In a CI/CD context, you'll want to run without this flag and instead provide credentials using environment variables instead.

Review the output to validate that your infrastructure changes have been deployed.

Finally, You can clean up any resources that were created by running Terrscale with the `--self-destroy` flag:

```bash
TF_VAR_pagerduty_token="your-pagerduty-token" \
TF_VAR_gcp_project_id="your-gcp-project-od" \
runiac deploy \
  -e nonprod \
  -a your-azure-subscription-id \
  --interactive \
  --container optumopensource/runiac:v0.0.1-beta3-alpine-azure-gcloud \
  --self-destroy
```
