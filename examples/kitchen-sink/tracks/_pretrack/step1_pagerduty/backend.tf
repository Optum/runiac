terraform {
  backend "local" {
    path = "pretrack/${var.runiac_step}/terraform.tfstate"
    workspace_dir = "/runiac/tfstate"
  }
}
