# Configure the Azure Provider
provider "google" {
  project         = var.account_id
  region          = var.region
  version         = "3.51.1"
}
