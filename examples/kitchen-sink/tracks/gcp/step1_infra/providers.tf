# Configure the Azure Provider
provider "google" {
  project         = var.gcp_project_id
  region          = var.region
  version         = "3.51.1"
}
