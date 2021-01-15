resource "google_storage_bucket" "example" {
  name          = "terrascale-example-bucket"
  location      = var.region
  force_destroy = true
  uniform_bucket_level_access = true
}
