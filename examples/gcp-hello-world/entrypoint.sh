#!/bin/sh

if gcloud auth application-default print-access-token ; then
  echo "already logged in to gcp..."
else
  gcloud auth application-default login
fi

gcloud config set project "$TERRASCALE_ACCOUNT_ID"


terrascale