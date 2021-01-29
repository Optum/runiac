#!/bin/bash

# Build builder images

push=false

if [ "$1" == "--push"  ]
then
  echo "Will push to dockerhub...."
  push=true
fi

rm -rf ./reports
outputVolume=$(docker volume create)
DOCKER_BUILDKIT=1 docker build -f "build/package/alpine-builder/Dockerfile" -t "runiac:alpine-builder" . || exit 1;
CID=$(docker create -v "$outputVolume":/reports "runiac:alpine-builder")
docker cp "$CID":/reports $(pwd)
touch ./reports/*.xml
docker rm "$CID"
docker volume rm "$outputVolume"


# Build consumer images
for d in build/package/*/ ; do
  if [[ "$d" == *"-builder"* ]]; then
    continue
  fi

  echo "$d"
  dir="${d%/*}"
  cleanDir=${dir##*/}

  # if not in github actions, set to local default
  if [ -z "$GITHUB_REF"  ]
  then
    GITHUB_REF=$(whoami)
  fi

  image="runiac:$GITHUB_REF-$cleanDir"
  DOCKER_BUILDKIT=1 docker build -f "$d/Dockerfile" -t "$image" . &

  if [ "$push" == "true"  ]
  then
    echo "pushing..."
    docker tag "$image" "optumopensource/$image"
    docker push "optumopensource/$image"
  fi

done

wait
