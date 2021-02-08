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

  # if not in version not set, set to local default
  if [ -z "$VERSION"  ]
  then
    VERSION=$(whoami)
  fi

  image="optumopensource/runiac:$VERSION-$cleanDir"
  DOCKER_BUILDKIT=1 docker build -f "$d/Dockerfile" -t "$image" . &

  if [ "$push" == "true"  ]
  then
    echo "pushing..."
    docker tag "$image" "optumopensource/$image" || exit 1
    docker push "optumopensource/$image"
  fi

done

wait
