# Configure the Azure Provider
provider "google" {
  project         = var.gcp_project_id
  region          = var.region
}

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 3.51.0"
    }
  }
}
