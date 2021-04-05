terraform {
  backend "local" {
    path = "gcp/${var.runiac_step}/terraform.tfstate"
    workspace_dir = "/runiac/tfstate"
  }
}
