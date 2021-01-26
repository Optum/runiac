resource azurerm_resource_group example {
  name     = var.resource_group
  location = var.region
}

//resource azurerm_storage_account example {
//  name                     = "struniacsample"
//  resource_group_name      = azurerm_resource_group.example.name
//  location                 = azurerm_resource_group.example.location
//  account_tier             = "Standard"
//  account_replication_type = "GRS"
//
//  tags = {
//    environment = "staging"
//  }
//}
