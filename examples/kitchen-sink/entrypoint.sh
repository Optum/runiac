#!/bin/sh

if az account get-access-token ; then
  echo "already logged in to azure..."
else
  az login;
fi

if gcloud auth application-default print-access-token ; then
  echo "already logged in to gcp..."
else
  gcloud auth application-default login
  gcloud config set account "$TF_VAR_gcp_project_id"
fi

terrascale
