terraform {
  backend "local" {
    path = "${var.runiac_step}.terraform.tfstate"
    workspace_dir = "/runiac/tfstate"
  }
}
