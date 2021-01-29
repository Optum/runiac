locals {
    runiac_namespace- = var.runiac_namespace == "" ? "" : "${var.runiac_namespace}-"
}

variable runiac_account_id {
    type = string
}

variable runiac_namespace {
    type = string
}

variable runiac_region {
    type = string
}

variable runiac_environment {
    type = string
}

variable runiac_step {
    type = string
}