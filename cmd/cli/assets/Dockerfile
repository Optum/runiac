# syntax = docker/dockerfile:experimental

ARG TERRASCALE_CONTAINER="terrascale:alpine-azure"

FROM golang:1.13 as builder

WORKDIR /app

RUN apt-get update && apt-get install -y unzip bash

RUN mkdir linux_amd64

SHELL ["/bin/bash", "-c"]

# Declare terraform providers and their versions here
RUN declare -A p; \
    p[random]=2.3.0; \
    p[archive]=1.2.2; \
    p[google]=3.51.1; \
    for provider in "${!p[@]}"; do version=${p[$provider]} && curl -o ${provider}.zip "https://releases.hashicorp.com/terraform-provider-${provider}/${version}/terraform-provider-${provider}_${version}_linux_amd64.zip" && unzip ${provider}.zip -d "/app/linux_amd64" && rm -f ${provider}.zip & done; wait;


FROM $TERRASCALE_CONTAINER

COPY --from=builder /app/linux_amd64/ /root/.terraform.d/plugins/linux_amd64

WORKDIR /app

COPY . .

RUN mkdir -p $HOME/.terraform.d/plugin-cache

COPY entrypoint.sh entrypoint.sh
RUN chmod +x entrypoint.sh


ENTRYPOINT ["bash", "-c", "./entrypoint.sh"]