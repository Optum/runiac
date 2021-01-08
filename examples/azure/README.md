# Basic Terrascale Examples for Azure

This example will provide a simple starting point for working with Terrascale and deploying resources into
a Microsoft Azure subscription.

The following steps assume you are running on a Linux or macOS system, but the process will most likely be similar on Windows.

## Requirements

You'll need an active Microsoft Azure subscription to run this example. Microsoft offers a [free tier](https://azure.microsoft.com/en-us/free/), which
can suffice for this example. Be sure to [install the Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) on your machine as well.

Once you have a subscription, you'll need to run `az login` and authenticate yourself against Azure. If successful, you should see
a list of subscriptions that you have access to. Set your default account by providing the subscription ID to the Azure CLI:

`az account set -s YOUR-SUBSCRIPTION-ID`

## Running

Assuming you've set up your subscription and login credentials, you can execute Terrascale using the following command:

```bash
./deploy.sh -a your-azure-subscription-id --dry-run
```

This will run Terrascale without commiting any infrastructure changes. You can view the output to see if it aligns with expectations. The example
creates a [resource group](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group) and an empty
[storage account](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account), but you can add more 
resources under the `steps` directory.

To deploy infrastructure changes, you can run the following command instead:

```bash
./deploy.sh -a your-azure-subscription-id
```

Once again, review the output to validate that your infrastructure changes have been deployed.

## Important Notes

This example assumes you are using your own login credentials to deploy infrastructure. In a real world situation, you most likely will
want to use a [service principal](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals), especially
if you intend to use Terrascale in an Azure CI/CD pipeline. A common approach is to use a service principal with a client ID and secret, as 
[described here](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_client_secret).

To do this, you'll need to modify the provided `deploy.sh`, and add the following four environment variables to the Docker container execution:

```bash
docker run \
  $volumeMap \
  -e ARM_CLIENT_ID="$ARM_CLIENT_ID" \
  -e ARM_CLIENT_SECRET="$ARM_CLIENT_SECRET" \
  -e ARM_TENANT_ID="$ARM_TENANT_ID" \
  -e ARM_SUBSCRIPTION_ID="$ACCOUNT_ID" \
  ...
```

In the context of an Azure YAML pipeline, you can obtain these values by setting the `addSpnToEnvironment` input to `true` on the 
[AzureCLI@2](https://docs.microsoft.com/en-us/azure/devops/pipelines/tasks/deploy/azure-cli?view=azure-devops) task.
