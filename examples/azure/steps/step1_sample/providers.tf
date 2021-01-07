# Configure the Azure Provider
provider "azurerm" {
  subscription_id = var.account_id
  version         = "2.28.0"
  features {}
}
