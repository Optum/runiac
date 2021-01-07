terraform {
  backend "local" {
    path = "/opt/tfstates/default.tfstate"
  }
}
