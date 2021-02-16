# Configure the Azure Provider
provider "azurerm" {
  subscription_id = var.runiac_account_id
  features {}
}

provider "pagerduty" {
  token = var.pagerduty_token
}

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 2.28.0"
    }
    
    pagerduty = {
      source  = "PagerDuty/pagerduty"
      version = "~> 1.8.0"
    }
  }
}