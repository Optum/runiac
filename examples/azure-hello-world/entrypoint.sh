#!/bin/sh

if az account get-access-token ; then
  echo "already logged in..."
else
  az login;
fi

runiac