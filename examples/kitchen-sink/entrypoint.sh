#!/bin/sh

if az account get-access-token ; then
  echo "already logged in to azure..."
else
  az login;
fi

if gcloud auth print-access-token ; then
  echo "already logged in to gcp..."
else
  gcloud auth application-default login;
fi

terrascale
