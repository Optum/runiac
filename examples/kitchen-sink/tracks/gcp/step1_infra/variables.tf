locals {
    docker_image = "gcr.io/cloudrun/hello"

    region = lookup({
        "centralus": "us-central1"
    }, var.region, "centralus")
}

variable pagerduty_token {
    type = string
}

variable pretrack-pagerduty-pagerduty_policy_id {
    type = string
}

variable gcp_project_id {
    type = string
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
