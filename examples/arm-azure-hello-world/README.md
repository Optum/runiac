# ARM Template Basic runiac Examples for Azure

This example will provide a simple starting point for working with runiac and deploying resources into
a Microsoft Azure subscription using ARM templates.

The following steps assume you are running on a Linux or macOS system, but the process will most likely be similar on Windows.

## Requirements

See the top-level README for information on obtaining these items:

- An Azure subscription

## Running

If you did not clone this repository, you can have runiac scaffold a copy of this example by running:

```bash
runiac new --url github.com/optum/runiac//examples/arm-azure-hello-world
```

Assuming you've set up your subscription and login credentials, you can execute runiac using the following command:

```bash
runiac -a <your-azure-subscription-id> --dry-run
```

This will run runiac without commiting any infrastructure changes. You can view the output to see if it aligns with expectations. The example
creates a [resource group](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group), and a storage
account using ARM templates. You can add more resources within the `step1_default` directory.

To deploy infrastructure changes, you can run the following command instead:

```bash
runiac -a <your-azure-subscription-id> -e <your-environment-name>
```

Review the output to validate that your infrastructure changes have been deployed.

Finally, You can clean up any resources that were created by running runiac with the `--self-destroy` flag:

```bash
runiac -a <your-azure-subscription-id> -e <your-environment-name> --self-destroy
```

## Important Notes

This example assumes you are using your own login credentials to deploy infrastructure. In a real world situation, you most likely will
want to use a [service principal](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals), especially
if you intend to use runiac in a CI/CD pipeline.

In the context of an Azure YAML pipeline, you can obtain these values by setting the `addSpnToEnvironment` input to `true` on the 
[AzureCLI@2](https://docs.microsoft.com/en-us/azure/devops/pipelines/tasks/deploy/azure-cli?view=azure-devops) task.
