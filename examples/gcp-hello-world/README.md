# Hello World runiac Project for Google Cloud Platform

This example will provide a starting point for working with runiac and deploying resources into
a Google Cloud Platform project.

The following steps assume you are running on a Linux or macOS system, but the process will be similar on Windows.

## Requirements

See the top-level README for information on obtaining these items:

- A Google Cloud Platform project

## Running

Assuming you've set up your service account credentials, you can execute runiac using the following command:

```bash
runiac apply -a <your-gcp-project-id> -e <your-environment-name> --dry-run
```

This will run runiac without committing any infrastructure changes. You can view the output to see if it aligns with expectations. The example
creates a [GCP Storage bucket](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket) in your project,
you can add more resources under the `step1_default` directory.

To deploy infrastructure changes, you can run the following command instead:

```bash
runiac -a <your-gcp-project-id> -e <your-environment-name>
```

Review the output to validate that your infrastructure changes have been deployed.

Finally, You can clean up any resources that were created by running Terrscale with the `--self-destroy` flag:

```bash
runiac -a <your-gcp-project-id> -e <your-environment-name> --self-destroy
```

## Important Notes

Be aware that some Google Cloud Platform resources are not deleted immediately. Common examples include [IAM roles](https://cloud.google.com/iam/docs/creating-custom-roles#deleting-custom-role), among others, which remain in the system for a period of time before finally being purged 
(ie: soft deletes). The Terraform provider documentation will usually [call this out](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/google_project_iam_custom_role) in a warning.

This has implications on ephemeral deployments; you cannot create a role with a given name, run runiac with the `--self-destroy` flag
in this example, and rerun runiac immediately afterwards. GCP will detect a conflict when the same role is created again, and as a result, your
deployment will fail.

For these types of resources, the recommendation is to only deploy them to non-ephemeral environments. You can leverage Terraform's `count` property
and runiac's `namespace` variable to conditionally deploy such resources:

```hcl-terraform
resource "google_project_iam_custom_role" "my-custom-role" {
  count = var.namespace != "" ? 0 : 1
}
```