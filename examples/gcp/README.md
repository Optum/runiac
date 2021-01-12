# Basic Terrascale Examples for Google Cloud Platform

This example will provide a simple starting point for working with Terrascale and deploying resources into
a Google Cloud Platform project.

The following steps assume you are running on a Linux or macOS system, but the process will most likely be similar on Windows.

## Requirements

You'll need a project on Google Cloud Platform where you can deploy resources. Google offers a [free tier](https://cloud.google.com/free) for
evaluation, which can suffice for this example.

After you've acquired a project, you'll need to create a [service account](https://cloud.google.com/iam/docs/service-accounts) with the necessary
IAM permissions on your project. You should generally only grant the minimal amount of permissions needed to deploy your infrastructure, and add
permissions as needed down the road. Follow [the Google documentation](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) 
for setting up service accounts, and make sure to create a key for your account once it's ready.

You can then download the service account key as a JSON file and place it in the root of this example directory. Save it as `credentials.json`.
This file will be copied to the Terrascale Docker image during the build process.

At this point, you should be all set in terms of required steps prior to deployment.

## Running

Assuming you've set up your service account credentials, you can execute Terrascale using the following command:

```bash
./deploy.sh -a your-gcp-project-id --dry-run
```

This will run Terrascale without commiting any infrastructure changes. You can view the output to see if it aligns with expectations. The example
creates a [GCP Storage bucket](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket) in your project, but
you can add more resources under the `steps` directory.

To deploy infrastructure changes, you can run the following command instead:

```bash
./deploy.sh -a your-gcp-project-id
```

Review the output to validate that your infrastructure changes have been deployed.

Finally, You can clean up any resources that were created by running Terrscale with the `--self-destroy` flag:

```bash
./deploy.sh -a your-gcp-project-id --self-destroy
```

## Important Notes

Be aware that some Google Cloud Platform resources are not deleted immediately. Common examples include [IAM roles](https://cloud.google.com/iam/docs/creating-custom-roles#deleting-custom-role), among others, which remain in the system for a period of time before finally being purged 
(ie: soft deletes). The Terraform provider documentation will usually [call this out](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/google_project_iam_custom_role) in a warning.

This has implications on ephemeral deployments; you cannot create a role with a given name, run Terrascale with the `--self-destroy` flag
in this example, and rerun Terrascale immediately afterwards. GCP will detect a conflict when the same role is created again, and as a result, your
deployment will fail.

For these types of resources, the recommendation is to only deploy them to non-ephemeral environments. You can leverage Terraform's `count` property
and Terrascale's `namespace` variable to conditionally deploy such resources:

```hcl-terraform
resource "google_project_iam_custom_role" "my-custom-role" {
  count = var.namespace != "" ? 0 : 1
  
  ...
}
```