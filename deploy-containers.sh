#!/bin/bash

ACCOUNTID="346166872260"
NAMESPACE=""
AWSPROFILE="GaiaDeploy"

# Args
while test $# -gt 0; do
    case "$1" in
        --namespace)
            shift
            NAMESPACE=$1
            shift
            ;;
        --account-id)
            shift
            ACCOUNTID=$1
            shift
            ;;
        --aws-profile)
            shift
            AWSPROFILE=$1
            shift
            ;;
          *)
            echo "$1 is not a recognized flag!"
            echo "example usage: ./deploy-containers.sh --namespace mgrose --account-id 1234324"
            exit 1;
            ;;
    esac
done
#
#if [ -z "$NAMESPACE" ]
#then
#      NAMESPACE="$(whoami)"
#fi

NAMESPACE_=$NAMESPACE
if [ -n "$NAMESPACE_" ]; then
  NAMESPACE_="$NAMESPACE-"
fi

if [ "$AWSPROFILE" != "static" ]; then
  aws-vault remove "$AWSPROFILE" --sessions-only
  unset AWS_VAULT;
  unset AWS_SECRET_ACCESS_KEY;
  unset AWS_ACCESS_KEY_ID;
  unset AWS_SECURITY_TOKEN;
  unset AWS_SESSION_TOKEN;

  if [ -z "$NAMESPACE_" ]; then
    export $(export AWS_DEFAULT_REGION=us-east-1 && aws-vault exec "$AWSPROFILE" -- env | grep ^AWS | xargs);
  else
    # This unfortunately needs to be here as we cannot use below assume role command with above aws-vault command
    # https://github.com/99designs/aws-vault/issues/455
    export $(export AWS_DEFAULT_REGION=us-east-1 && aws-vault exec "GaiaDeployLocal" -- env | grep ^AWS | xargs);
  fi
else
  # If relevant, utilize the ephemeral role to authenticate to ecr
  creds=$(aws sts assume-role --role-arn "arn:aws:iam::$ACCOUNTID:role/${NAMESPACE_}GaiaDeployRole" --role-session-name "${NAMESPACE_}Gaia-Deploy-$ACCOUNTID" --output json | jq '{ accessKeyId: .Credentials.AccessKeyId, secretAccessKey: .Credentials.SecretAccessKey, sessionToken: .Credentials.SessionToken }') &>/dev/null

  export AWS_ACCESS_KEY_ID=$(echo "$creds" | jq -r '.accessKeyId') &>/dev/null
  export AWS_SECRET_ACCESS_KEY=$(echo "$creds" | jq -r '.secretAccessKey' ) &>/dev/null
  export AWS_SESSION_TOKEN=$(echo "$creds" | jq -r '.sessionToken') &>/dev/null
fi

# check for AWS CLI v2+
if [ $(aws ecr get-login-password 2>&1 | grep 'Invalid choice' | wc -l) == 0 ]; then
  aws ecr get-login-password | docker login --username AWS --password-stdin https://"$ACCOUNTID".dkr.ecr.us-east-1.amazonaws.com
elif [ $(aws ecr get-login 2>&1 | grep 'Invalid choice' | wc -l) == 0 ]; then
  $(aws ecr get-login --region us-east-1 --no-include-email) || { echo 'ERROR: aws ecr login failed' ; exit 1; };
fi

sh ./scripts/docker-build-push.sh "$ACCOUNTID" "us-east-1" "." "$NAMESPACE"
