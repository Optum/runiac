# Configure the Azure Provider
provider "azurerm" {
  subscription_id = var.runiac_account_id
  features {}
}

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 2.28.0"
    }
  }
}