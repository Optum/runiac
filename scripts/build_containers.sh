#!/bin/sh

for d in build/package/*/ ; do
  echo "$d"
  dir="${d%/*}"
  tag=${dir##*/}
  DOCKER_BUILDKIT=1 docker build -f "$d/Dockerfile" -t "terrascale:$tag" . &
done

wait
