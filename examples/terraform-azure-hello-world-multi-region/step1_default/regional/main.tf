resource "azurerm_resource_group" "spoke" {
  name     = "${local.namespace-}rg-runiac-spoke-${var.runiac_region}"
  location = var.runiac_region
}
