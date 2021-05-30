#!/bin/sh

if [[ -z "$ARM_CLIENT_ID" || -z "$ARM_CLIENT_SECRET" || -z "$ARM_TENANT_ID"  || -z "$ARM_SUBSCRIPTION_ID" ]]; then
  if az account get-access-token ; then
    echo "already logged in..."
  else
    az login || exit 1;
  fi
else
  az login --service-principal --username "$ARM_CLIENT_ID" --password "$ARM_CLIENT_SECRET" --tenant "$ARM_TENANT_ID"
fi

runiac
