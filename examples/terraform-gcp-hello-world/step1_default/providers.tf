# Configure the Azure Provider
provider "google" {
  project         = var.runiac_account_id
  region          = var.runiac_region
}
