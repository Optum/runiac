terraform {
  backend "local" {
    path = "/tfstate/gcp/${var.terrascale_step}/terraform.tfstate"
  }
}
