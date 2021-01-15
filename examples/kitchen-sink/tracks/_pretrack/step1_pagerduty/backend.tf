terraform {
  backend "local" {
    path = "/tfstate/pretrack/${var.terrascale_step}/terraform.tfstate"
  }
}
