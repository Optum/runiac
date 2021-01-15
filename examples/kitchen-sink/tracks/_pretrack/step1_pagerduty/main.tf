// deploy a PagerDuty team
resource "pagerduty_team" "example" {
  name        = "Engineering"
  description = "All engineering" 
}

// deploy a user account into PagerDuty
resource "pagerduty_user" "example" {
  name  = "John Smith"
  email = "john.smith.foo@getnada.com"
}

// associate the user with the team
resource "pagerduty_team_membership" "example" {
  user_id = pagerduty_user.example.id
  team_id = pagerduty_team.example.id
  role    = "manager"
}

// deploy an escalation policy
resource "pagerduty_escalation_policy" "example" {
  name      = "Engineering Escalation Policy"
  num_loops = 2

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user"
      id   = pagerduty_user.example.id
    }
  }
}
