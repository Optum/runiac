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
    p[google]=3.51.1; \
    for provider in "${!p[@]}"; do version=${p[$provider]} && curl -o ${provider}.zip "https://releases.hashicorp.com/terraform-provider-${provider}/${version}/terraform-provider-${provider}_${version}_linux_amd64.zip" && unzip ${provider}.zip -d "/app/linux_amd64" && rm -f ${provider}.zip & done; wait;

COPY ./steps ./steps

FROM $TERRASCALE_CONTAINER

COPY --from=builder /app/linux_amd64/ /root/.terraform.d/plugins/linux_amd64

ARG GCLOUD_CLI_VERSION=322.0.0

# install google cloud sdk
RUN apk add --no-cache python3
RUN cd /opt && \
    curl https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz > gcloud.tar.gz && \
    tar xfz gcloud.tar.gz && \
    ./google-cloud-sdk/install.sh

ENV PATH="/opt/google-cloud-sdk/bin:${PATH}"

WORKDIR /app

COPY --from=builder /app/steps /app/steps
COPY credentials.json /app/credentials.json
ENV GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json

RUN mkdir -p $HOME/.terraform.d/plugin-cache

ENTRYPOINT ["terrascale"]
