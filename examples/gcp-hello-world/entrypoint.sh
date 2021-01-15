#!/bin/sh

if gcloud auth application-default print-access-token ; then
  echo "already logged in to gcp..."
else
  gcloud auth application-default login
  gcloud config set account "$TF_VAR_gcp_project_id"
fi

terrascale