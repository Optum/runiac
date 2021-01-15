terraform {
  backend "local" {
    path = "/tfstate/azure/${var.terrascale_step}/${var.region}/terraform.tfstate"
  }
}
