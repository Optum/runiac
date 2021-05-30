resource azurerm_resource_group hub {
  name     = "${local.namespace-}rg-runiac-hub-${var.runiac_region}"
  location = var.runiac_region
}