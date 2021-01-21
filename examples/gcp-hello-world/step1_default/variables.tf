locals {
    terrascale_namespace- = var.namespace == "" ? "" : "${var.namespace}-"
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

variable terrascale_step {
    type = string
}