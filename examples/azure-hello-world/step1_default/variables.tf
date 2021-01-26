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
    default = "rg-runiac-sample"
}

variable runiac_step {
    type = string
}