# syntax = docker/dockerfile:experimental

ARG GOVERSION=1.15
 
FROM terrascale:alpine-builder as builder

FROM hashicorp/terraform:0.14.4

RUN apk update

# Common tools
RUN apk add bash \
    && apk add jq \
    && apk add curl \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*

ARG GCLOUD_CLI_VERSION=322.0.0

# install google cloud sdk
RUN apk add --no-cache python3
RUN cd /opt && \
    curl https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_CLI_VERSION}-linux-x86_64.tar.gz > gcloud.tar.gz && \
    tar xfz gcloud.tar.gz && \
    ./google-cloud-sdk/install.sh

ENV PATH="/opt/google-cloud-sdk/bin:${PATH}"

RUN mkdir -p $HOME/.terraform.d/plugins/linux_amd64
RUN mkdir -p $HOME/.terraform.d/plugin-cache

# Grab from builder
COPY --from=builder /app/terrascale /usr/local/bin
COPY --from=builder /usr/local/bin/test2json /usr/local/bin/test2json
COPY --from=builder /usr/local/bin/gotestsum /usr/local/bin/gotestsum

ENV TF_IN_AUTOMATION true
ENV GOVERSION ${GOVERSION} # https://github.com/gotestyourself/gotestsum/blob/782abf290e3d93b9c1a48f9aa76b70d65cae66ed/internal/junitxml/report.go#L126

ENTRYPOINT [ "terrascale" ]
