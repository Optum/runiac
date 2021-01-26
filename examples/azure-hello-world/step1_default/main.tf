resource azurerm_resource_group example {
  name     = "${local.namespace-}${var.resource_group}"
  location = var.runiac_region
}