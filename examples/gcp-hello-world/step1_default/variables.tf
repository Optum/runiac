locals {
    runiac_namespace- = var.namespace == "" ? "" : "${var.namespace}-"
}

variable account_id {
    type = string
}

variable namespace {
    type = string
}

variable region {
    type = string
}

variable environment {
    type = string
}

variable runiac_step {
    type = string
}