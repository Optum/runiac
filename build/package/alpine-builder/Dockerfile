# syntax=docker/dockerfile:1

ARG GOVERSION=1.25.7

FROM golang:${GOVERSION} AS builder

RUN apt-get update && apt-get upgrade -y ca-certificates && apt-get install -y bash && apt-get install -y unzip

RUN curl -L -o gotestsum.tgz "https://github.com/gotestyourself/gotestsum/releases/download/v1.13.0/gotestsum_1.13.0_linux_amd64.tar.gz" && \
    tar -C /usr/local/bin -xzf gotestsum.tgz && \
    rm gotestsum.tgz && \
    rm -f /usr/local/bin/LICENSE && \
    rm -f /usr/local/bin/README.md;

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

FROM hashicorp/terraform:1.14.6

# Common tools
RUN apk add --no-cache \
    bash \
    jq \
    curl \
    ca-certificates

RUN mkdir -p $HOME/.terraform.d/plugin-cache

# Grab from builder
COPY --from=builder /app/runiac /usr/local/bin
COPY --from=builder /usr/local/bin/gotestsum /usr/local/bin/gotestsum

ENV TF_IN_AUTOMATION=true
# Required by gotestsum JUnit XML report:
# https://github.com/gotestyourself/gotestsum/blob/782abf290e3d93b9c1a48f9aa76b70d65cae66ed/internal/junitxml/report.go#L126
ENV GOVERSION=${GOVERSION}

ENTRYPOINT [ "runiac" ]
