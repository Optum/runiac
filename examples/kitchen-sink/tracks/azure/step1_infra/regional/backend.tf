terraform {
  backend "local" {
    path = "/tfstate/azure/${var.runiac_step}/${var.region}/regional/terraform.tfstate"
  }
}
