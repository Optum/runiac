# Configure the Azure Provider
provider "pagerduty" {
  token = var.pagerduty_token
}

terraform {
  required_providers {
    pagerduty = {
      source  = "PagerDuty/pagerduty"
      version = "~> 1.8.0"
    }
  }
}