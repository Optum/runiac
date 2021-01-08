#!/bin/bash

set -e
set -o pipefail

pushd $(pwd) > /dev/null
cd "${0%/*}"

# defaults
DRY_RUN="false"
SELF_DESTROY="false"
ENVIRONMENT="local"
DEPLOYMENT_RING="local"
ACCOUNT_ID=""
VERSION=""
LOG_LEVEL="info"
INTERACTIVE_FLAG="-it"

# arguments
while [[ "$1" =~ ^- && ! "$1" == "--" ]]; do case $1 in
  -a | --account )
    shift; ACCOUNT_ID=$1
    ;;
  -c | --terrascale-container )
    shift; BASE_CONTAINER=$1
    ;;
  -e | --environment )
    shift; ENVIRONMENT=$1
    ;;
  -r | --deployment-ring )
    shift; DEPLOYMENT_RING=$1
    ;;
  --pull-request-id )
    shift; NAMESPACE="pr-$1"; DEPLOYMENT_RING="pr";
    ;;
  --log-level )
    shift; LOG_LEVEL=$1
    ;;
  --dry-run )
    DRY_RUN="true"
    ;;
  --self-destroy )
    SELF_DESTROY="true"
    ;;
  --non-interactive )
    INTERACTIVE_FLAG=""
    ;;
esac; shift; done
if [[ "$1" == '--' ]]; then shift; fi

# check for required arguments, and set defaults sensibly
if [ -z "$ACCOUNT_ID" ]; then echo "-a or --account is required"; exit 1; fi
if [ -z "$ENVIRONMENT" ]; then echo "-e or --environment is required"; exit 1; fi

# set sensible defaults for local development
if [ "$ENVIRONMENT" = "local" ]; then 
  NAMESPACE="$(whoami)"
  VERSION="$(whoami)"
fi

# list which steps to execute
TRACK_STEPS=(
  "#terrascale#sample"
)

STEP_WHITELIST=$(printf ",%s", "${TRACK_STEPS[@]}")
STEP_WHITELIST=${STEP_WHITELIST:1}

DOCKER_BUILDKIT=1 docker build \
  -t tssample \
  --build-arg TERRASCALE_CONTAINER="$BASE_CONTAINER" .

docker run \
  -e VERSION="$VERSION" \
  -e CSP="GCP" \
  -e ACCOUNT_ID="$ACCOUNT_ID" \
  -e ENVIRONMENT="$ENVIRONMENT" \
  -e NAMESPACE="$NAMESPACE" \
  -e TERRASCALE_REGION_GROUP="us" \
  -e TERRASCALE_STEP_WHITELIST="$STEP_WHITELIST" \
  -e TERRASCALE_DRY_RUN="$DRY_RUN" \
  -e TERRASCALE_SELF_DESTROY="$SELF_DESTROY" \
  -e LOG_LEVEL="$LOG_LEVEL" \
  -e DEPLOYMENT_RING="$DEPLOYMENT_RING" \
  $INTERACTIVE_FLAG tssample
