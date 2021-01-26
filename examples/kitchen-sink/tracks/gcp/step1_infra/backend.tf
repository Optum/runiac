terraform {
  backend "local" {
    path = "/tfstate/gcp/${var.runiac_step}/terraform.tfstate"
  }
}
