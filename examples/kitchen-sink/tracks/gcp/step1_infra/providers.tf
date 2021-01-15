# Configure the Azure Provider
provider "google" {
  project         = var.gcp_project_id
  region          = var.region
}

provider "pagerduty" {
  token = var.pagerduty_token
}

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 3.51.0"
    }
    
    pagerduty = {
      source  = "PagerDuty/pagerduty"
      version = "~> 1.8.0"
    }
  }
}
