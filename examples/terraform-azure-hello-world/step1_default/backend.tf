terraform {
  backend "local" {
    path = "/runiac/tfstate/${var.runiac_step}/terraform.tfstate"
  }
}
