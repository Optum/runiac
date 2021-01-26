#!/bin/bash

# Build builder images
DOCKER_BUILDKIT=1 docker build -f "build/package/alpine-builder/Dockerfile" -t "runiac:alpine-builder" . || exit 1

# Build consumer images
for d in build/package/*/ ; do
  if [[ "$d" == *"-builder"* ]]; then
    continue
  fi

  echo "$d"
  dir="${d%/*}"
  tag=${dir##*/}
  DOCKER_BUILDKIT=1 docker build -f "$d/Dockerfile" -t "runiac:$tag" . &
done

wait
