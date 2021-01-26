#!/bin/sh

if gcloud auth application-default print-access-token ; then
  echo "already logged in to gcp..."
else
  gcloud auth application-default login
  gcloud config set project "$RUNIAC_ACCOUNT_ID"
fi



runiac