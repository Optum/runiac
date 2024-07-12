terraform {
  backend "local" {
    path          = "azure/${var.runiac_step}/terraform.tfstate"
    workspace_dir = "/runiac/tfstate"
  }
}
