resource "azurerm_resource_group" "example" {
  name     = "${var.resource_group}-${var.region}"
  location = var.region
}

resource "azurerm_app_service_plan" "example" {
  name                = "plan-terrascale-sample-${var.region}"
  location            = var.region
  resource_group_name = azurerm_resource_group.example.name
  kind                = "Linux"
  reserved            = true

  sku {
    tier = "Standard"
    size = "S1"
  }
}

resource "azurerm_app_service" "example" {
  name                = "appsvc-terrascale-example-${var.region}"
  location            = var.region
  resource_group_name = azurerm_resource_group.example.name
  app_service_plan_id = azurerm_app_service_plan.example.id

  site_config {
    app_command_line = ""
    linux_fx_version = "DOCKER|${local.docker_image}"
    always_on        = true
  }
}
