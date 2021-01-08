terraform {
  backend "local" {
    path = "/opt/tfstate/default.tfstate"
  }
}
