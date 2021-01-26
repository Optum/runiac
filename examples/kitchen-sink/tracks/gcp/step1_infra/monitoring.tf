resource "pagerduty_service" "example" {
  name                    = "cr-runiac-sample-${var.region}"
  auto_resolve_timeout    = 14400
  acknowledgement_timeout = 600
  escalation_policy       = var.pretrack-pagerduty-pagerduty_policy_id
  alert_creation          = "create_alerts_and_incidents"
}

resource "pagerduty_service_integration" "example" {
  name    = "Generic integration - cr-runiac-sample-${var.region}"
  type    = "generic_events_api_inbound_integration"
  service = pagerduty_service.example.id
}
