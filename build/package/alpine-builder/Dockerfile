# syntax=docker/dockerfile:1

ARG GOVERSION=1.26.3
ARG TERRAFORM_VERSION=1.14.9

FROM golang:${GOVERSION} AS builder

RUN apt-get update && apt-get upgrade -y ca-certificates && apt-get install -y bash

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go install gotest.tools/gotestsum@v1.13.0

WORKDIR /app

RUN mkdir /reports

COPY go.mod ./
COPY go.sum ./

COPY pkg ./pkg
COPY cmd ./cmd
COPY plugins ./plugins

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    gotestsum --format standard-verbose --junitfile ./reports/junit.xml --raw-command -- go test -parallel 5 --json ./... || echo "failed"

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ./runiac ./cmd/runiac/

FROM hashicorp/terraform:${TERRAFORM_VERSION}

ARG GOVERSION
ARG TERRAFORM_VERSION

# Common tools
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    bash \
    jq \
    curl \
    ca-certificates

RUN mkdir -p $HOME/.terraform.d/plugin-cache

# Grab from builder
COPY --from=builder /app/runiac /usr/local/bin
COPY --from=builder /go/bin/gotestsum /usr/local/bin/gotestsum

ENV TF_IN_AUTOMATION=true
# Required by gotestsum JUnit XML report:
# https://github.com/gotestyourself/gotestsum/blob/782abf290e3d93b9c1a48f9aa76b70d65cae66ed/internal/junitxml/report.go#L126
ENV GOVERSION=${GOVERSION}

ENTRYPOINT [ "runiac" ]
