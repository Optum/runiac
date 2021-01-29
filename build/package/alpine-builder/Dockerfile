# syntax = docker/dockerfile:experimental

ARG GOVERSION=1.15
 
FROM golang:${GOVERSION} as builder

RUN apt-get update && apt-get upgrade -y ca-certificates && apt-get install -y bash && apt-get install -y unzip

RUN curl -Lo go.zip "https://github.com/golang/go/archive/go1.15.1.zip" && \
    unzip go.zip && \
    rm -f go.zip && \
    cd go-go1.15.1/src/cmd/test2json/ && \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" . && \
    mv test2json /usr/local/bin/test2json && \
    rm -rf /go-go1.15.1

RUN curl -L -o gotestsum.tgz "https://github.com/gotestyourself/gotestsum/releases/download/v0.5.2/gotestsum_0.5.2_linux_amd64.tar.gz" && \
    tar -C /usr/local/bin -xzf gotestsum.tgz && \
    rm gotestsum.tgz && \
    rm /usr/local/bin/LICENSE && \
    rm /usr/local/bin/README.md;

WORKDIR /app

RUN mkdir /reports

COPY go.mod ./
COPY go.sum ./

COPY pkg ./pkg
COPY cmd ./cmd
COPY plugins ./plugins

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    gotestsum --format standard-verbose --junitfile /reports/junit.xml --raw-command -- go test -parallel 5 --json ./... || echo "failed"

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ./runiac ./cmd/runiac/

