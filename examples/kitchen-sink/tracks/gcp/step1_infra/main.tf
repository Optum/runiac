// deploy a storage bucket
resource "google_storage_bucket" "example" {
  name          = "terrascale-example-bucket-${local.region}"
  location      = local.region
  force_destroy = true

  uniform_bucket_level_access = true
}

// deploy a docker image to a cloud run instance
resource "google_cloud_run_service" "example" {
  name     = "cr-terrascale-sample-${local.region}"
  location = local.region

  template {
    spec {
      containers {
        image = local.docker_image
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}
