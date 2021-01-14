terraform {
  backend "local" {
    path = "/tfstate/${var.terrascale_step}/terraform.tfstate"
  }
}
