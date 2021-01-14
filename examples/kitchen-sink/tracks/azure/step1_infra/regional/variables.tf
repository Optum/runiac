locals {
    docker_image = "heroku/nodejs-hello-world:latest"
}

variable account_id {
    type = string
}

variable region {
    type = string
}

variable environment {
    type = string
}

variable resource_group {
    type = string
    default = "rg-terrascale-sample"
}

variable terrascale_step {
    type = string
}