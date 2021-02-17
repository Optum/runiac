locals {
    docker_image = "gcr.io/cloudrun/hello"

    region = lookup({
        "centralus": "us-central1"
    }, var.runiac_region, "centralus")
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
