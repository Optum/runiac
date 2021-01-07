# syntax = docker/dockerfile:experimental

ARG TERRASCALE_CONTAINER="terrascale"

FROM golang:1.13 as builder

WORKDIR /app

RUN apt-get update && apt-get install -y unzip bash

RUN mkdir linux_amd64 

SHELL ["/bin/bash", "-c"]

# Declare terraform providers and their versions here
RUN declare -A p; \
    p[random]=2.3.0; \
    p[archive]=1.2.2; \
    p[azurerm]=2.28.0; \
    for provider in "${!p[@]}"; do version=${p[$provider]} && curl -o ${provider}.zip "https://releases.hashicorp.com/terraform-provider-${provider}/${version}/terraform-provider-${provider}_${version}_linux_amd64.zip" && unzip ${provider}.zip -d "/app/linux_amd64" && rm -f ${provider}.zip & done; wait;

COPY ./steps ./steps

FROM $TERRASCALE_CONTAINER

COPY --from=builder /app/linux_amd64/ /root/.terraform.d/plugins/linux_amd64

ARG AZURE_CLI_VERSION=2.13.0

# install azure cli
RUN \
    apk update && \
    apk add --no-cache curl tar openssl sudo bash jq python3 py-pip && \
    apk add --virtual=build gcc libffi-dev musl-dev openssl-dev make python3-dev && \
    python3 -m pip --no-cache-dir install azure-cli==${AZURE_CLI_VERSION} && \
    apk del --purge build

WORKDIR /app

COPY --from=builder /app/steps /app/steps

RUN mkdir -p $HOME/.terraform.d/plugin-cache

ENTRYPOINT ["terrascale"]
