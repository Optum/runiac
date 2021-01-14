# Configure the Azure Provider
provider "azurerm" {
  subscription_id = var.account_id
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