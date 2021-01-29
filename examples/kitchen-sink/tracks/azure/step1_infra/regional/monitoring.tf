data "pagerduty_vendor" "azure" {
  name = "Azure"
}

resource "pagerduty_service" "example" {
  name                    = "appsvc-runiac-example-${var.region}"
  auto_resolve_timeout    = 14400
  acknowledgement_timeout = 600
  escalation_policy       = var.pretrack-pagerduty-pagerduty_policy_id
  alert_creation          = "create_alerts_and_incidents"
}

resource "pagerduty_service_integration" "example" {
  name    = data.pagerduty_vendor.azure.name
  service = pagerduty_service.example.id
  vendor  = data.pagerduty_vendor.azure.id
}
