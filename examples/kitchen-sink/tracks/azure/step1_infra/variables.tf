locals {
  docker_image = "heroku/nodejs-hello-world"
}

variable "runiac_account_id" {
  type = string
}

variable "runiac_region" {
  type = string
}

variable "runiac_environment" {
  type = string
}

variable "resource_group" {
  type    = string
  default = "rg-runiac-sample"
}

variable "runiac_step" {
  type = string
}