locals {
    docker_image = "heroku/nodejs-hello-world:latest"
}

variable pagerduty_token {
    type = string
}

variable pretrack-pagerduty-pagerduty_policy_id {
    type = string
}

variable runiac_account_id {
    type = string
}

variable runiac_region {
    type = string
}

variable runiac_environment {
    type = string
}

variable resource_group {
    type = string
    default = "rg-runiac-sample"
}

variable runiac_step {
    type = string
}

variable "runiac_primary_region" {
    type = string
}