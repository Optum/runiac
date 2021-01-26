# syntax = docker/dockerfile:experimental

ARG http_proxy
ARG https_proxy
ARG GOVERSION=1.15
 
FROM runiac:alpine-builder as builder

FROM hashicorp/terraform:0.14.4

RUN apk update

# Common tools
RUN apk add bash \
    && apk add jq \
    && apk add curl \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*

ARG AZURE_CLI_VERSION=2.13.0

# install azure cli
RUN \
    apk update && \
    apk add --no-cache curl tar openssl sudo bash jq python3 py-pip && \
    apk add --virtual=build gcc libffi-dev musl-dev openssl-dev make python3-dev && \
    python3 -m pip --no-cache-dir install azure-cli==${AZURE_CLI_VERSION} && \
    apk del --purge build


RUN mkdir -p $HOME/.terraform.d/plugins/linux_amd64
RUN mkdir -p $HOME/.terraform.d/plugin-cache

# Grab from builder
COPY --from=builder /app/runiac /usr/local/bin
COPY --from=builder /usr/local/bin/test2json /usr/local/bin/test2json
COPY --from=builder /usr/local/bin/gotestsum /usr/local/bin/gotestsum

ENV TF_IN_AUTOMATION true
ENV GOVERSION ${GOVERSION} # https://github.com/gotestyourself/gotestsum/blob/782abf290e3d93b9c1a48f9aa76b70d65cae66ed/internal/junitxml/report.go#L126

ENTRYPOINT [ "runiac" ]
