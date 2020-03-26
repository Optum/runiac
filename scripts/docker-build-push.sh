#!/bin/bash

# ARGS
ACCOUNT_ID=$1
REGION=$2               # Region
DOCKER_CONTEXT=$3       # The path to the Dockfile from the root of the customer directory
NAMESPACE=$4            # The name for this run, e.g. pr-1 or mgrose

set -euxo pipefail

cd "$DOCKER_CONTEXT"
pwd

image_version=$(grep version version.json | awk -F'"' '{print $4}')
BASE_ECR_REPO_NAME=$(grep repo_name version.json | awk -F'"' '{print $4}')

ecr_repo_name=""
# Determine the ecr repo name and url
if [ -n "$NAMESPACE" ]; then
  ecr_repo_name="$BASE_ECR_REPO_NAME"
  image_version="${NAMESPACE}-${image_version}"
else
  ecr_repo_name="$BASE_ECR_REPO_NAME"
fi

ecr_repo_uri="$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com/$ecr_repo_name"
image_tag="$ecr_repo_name:$image_version"

echo "Building $DOCKER_CONTEXT for $ecr_repo_uri"

rm -rf ./reports
outputVolume=$(docker volume create)
DOCKER_BUILDKIT=1 docker build  --target builder -t "$ecr_repo_name" .;
CID=$(docker create -v "$outputVolume":/reports "$ecr_repo_name")
docker cp "$CID":/reports $(pwd)
touch ./reports/*.xml
docker rm "$CID"
docker volume rm "$outputVolume"

echo "Checking if the image exists in $ecr_repo_name..."
echo $(aws ecr list-images --registry-id $ACCOUNT_ID --repository-name $ecr_repo_name --output json | jq .imageIds) > images.json
for tag in $(jq -r ".[] | .imageTag" images.json)
do
  echo "Found tag: $tag"
  if [ "$tag" == "$image_version" ]; then
    if [ -n "$NAMESPACE" ]; then
      # Remove the already tagged image for PRs
      aws ecr batch-delete-image --registry-id "$ACCOUNT_ID" --repository-name "$ecr_repo_name" --image-ids imageTag="$image_version"
      rm -rf images.json
    else
      echo "$image_version tag already exists for $ecr_repo_name. Skipping build and push."
      rm -rf images.json
      exit 0
    fi
  fi
done
rm -rf images.json

echo "Pushing image to: $ecr_repo_uri"

echo "Image with tag: $image_version did not exist. Building and pushing..."
# shellcheck disable=SC2086
DOCKER_BUILDKIT=1 docker build --progress plain -t "$image_tag" .
docker tag "$ecr_repo_name":"$image_version" "$ecr_repo_uri":"$image_version"
docker push "$ecr_repo_uri":"$image_version"
