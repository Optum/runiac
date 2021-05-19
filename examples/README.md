<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [runiac Examples](#runiac-examples)
  - [Index](#index)
  - [Common Requirements](#common-requirements)
    - [Azure](#azure)
    - [Google Cloud Platform](#google-cloud-platform)
    - [PagerDuty](#pagerduty)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# runiac Examples

This directory contains several example projects that you can use as a starting point for runiac.

These examples can be individually scaffolded by using the runiac CLI. See each of the specific READMEs for more information.

## Index

* [arm-azure-hello-world](arm-azure-hello-world/): deploy ARM template infrastructure to Microsoft Azure
* [terraform-azure-hello-world](terraform-azure-hello-world/): deploy Terraform infrastructure to Microsoft Azure
* [terraform-gcp-hello-world](terraform-gcp-hello-world/): deploy Terraform infrastructure to Google Cloud Platform
* [kitchen-sink](kitchen-sink): deploy complex Terraform infrastructure simultaneously to both Microsoft Azure and Google Cloud Platform

## Common Requirements

Many of these examples will require a valid cloud provider account. Depending on the example you intend to try,
see the below documentation for tips making sure you have the necessary tooling and access in place.

### Azure

You'll need an active Microsoft Azure subscription to run this example. Microsoft offers a [free tier](https://azure.microsoft.com/en-us/free/), which
can suffice for this example. Be sure to [install the Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) on your machine as well.

Once you have a subscription, you'll need to run `az login` and authenticate yourself against Azure. If successful, you should see
a list of subscriptions that you have access to. Set your default account by providing the subscription ID to the Azure CLI:

`az account set -s YOUR-SUBSCRIPTION-ID`

### Google Cloud Platform

You'll need a project on Google Cloud Platform where you can deploy resources. Google offers a [free tier](https://cloud.google.com/free) for
evaluation, which can suffice for this example.

After you've acquired a project, you'll need to create a [service account](https://cloud.google.com/iam/docs/service-accounts) with the necessary
IAM permissions on your project. You should generally only grant the minimal amount of permissions needed to deploy your infrastructure, and add
permissions as needed down the road. Follow [the Google documentation](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) 
for setting up service accounts, and make sure to create a key for your account once it's ready.

You can then download the service account key as a JSON file and place it in the root of this example directory. Save it as `credentials.json`.
This file will be copied to the runiac Docker image during the build process.

At this point, you should be all set in terms of required steps prior to deployment.

### PagerDuty

You'll need a PagerDuty account where you have administrative permissions. PagerDuty offers a [free license](https://www.pagerduty.com/sign-up-free/?type=free) to try their service. 

After logging in, create a new API key from the Developer Tools -> API Access menu on the top-right. Copy this API key somewhere safe,
since it cannot be viewed again unless you create a new one.
