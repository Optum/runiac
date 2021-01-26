resource google_storage_bucket example {
  // namespace is used to enable local ("--local") deployment ring configurations, specific to the executors machine
  name          = "${local.runiac_namespace-}runiac-example-bucket"
  force_destroy = true
  uniform_bucket_level_access = true
}
