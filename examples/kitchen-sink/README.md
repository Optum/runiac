# Advanced Terrascale Example

This example will provide a more complex deployment scenario involving multiple cloud providers and services.
* A pre-track that deploys PagerDuty monitoring resources
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

Assuming you've set up your subscription and login credentials, you can execute Terrascale using the following Terrascale CLI command:

```bash
TF_VAR_pagerduty_token="your-pagerduty-token" \
TF_VAR_gcp_project_id="your-gcp-project-od" \
terrascale apply \
  -e nonprod \
  -a your-azure-subscription-id \
  --base-container "terrascale:azure-azure-gcloud" \
  --dry-run
```

This will run Terrascale without commiting any infrastructure changes. You can view the output to see if it aligns with expectations. The example
creates a [resource group](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group) and an empty
[storage account](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account), but you can add more 
resources under the `steps` directory.

To deploy infrastructure changes, you can run the following command instead:

```bash
TF_VAR_pagerduty_token="YOUR_PG_TOKEN" terrascale apply -e nonprod -a your-azure-subscription-id --base-container "terrascale:azure-gcloud"
```

Review the output to validate that your infrastructure changes have been deployed.

Finally, You can clean up any resources that were created by running Terrscale with the `--self-destroy` flag:

```bash
TF_VAR_pagerduty_token="YOUR_PG_TOKEN" terrascale apply -e nonprod -a your-azure-subscription-id --base-container "terrascale:azure-gcloud" --self-destroy
```
