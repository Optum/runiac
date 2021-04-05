terraform {
  backend "local" {
    path = "default.tfstate"
    workspace_dir = "/runiac/tfstate"
  }
}
