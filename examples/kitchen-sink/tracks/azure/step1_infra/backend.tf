terraform {
  backend "local" {
    path = "/tfstate/azure/${var.runiac_step}/${var.runiac_region}/terraform.tfstate"
  }
}
