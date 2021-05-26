# runiac - Run IaC Anywhere With Ease

[Documentation](https://runiac.io/docs)

[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://GitHub.com/optum/runiac/graphs/commit-activity)
![build](https://github.com/optum/runiac/workflows/build/badge.svg?branch=main)
[![Github all releases](https://img.shields.io/github/downloads/optum/runiac/total.svg)](https://GitHub.com/optum/runiac/releases/)

[comment]: <> (<a href="https://cla-assistant.io/Optum/runiac"><img src="https://cla-assistant.io/readme/badge/Optum/runiac" alt="CLA assistant" /></a>)

[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)

![](./logo.jpg)

---

A tool for running infrastructure as code (e.g. Terraform) anywhere with ease.

- Ability to change and test infrastructure changes locally with a production like environment
- Ability to make infrastructure changes without making pipeline changes
- Quality developer experience
- Container-based, execute anywhere and on any CI/CD system
- Multi-Region deployments built-in
- Handling groups of regions for data privacy regulations
- Enabling "terraservices"
- Keeping Your Pipelines Simple
- Plugin-based

> NOTE: README documentation is out of date and will be removed soon. Please see [runiac.io](https://runiac.io) for latest docs

[comment]: <> (runiac is meant to be run as an image. We do **not** recommend running the `runiac` executor binary in another image, as it might not work.)

We'd love to hear from you! Submit github issues for questions, issues or feedback.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [How does runiac work?](#how-does-runiac-work)
- [Demo](#demo)
- [Install](#install)
- [Tutorial](#tutorial)
- [Using runiac](#using-runiac)
  - [Inputs](#inputs)
- [Contributing](#contributing)
  - [Running Locally](#running-locally)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Demo

See runiac in action on [runiac.io](https://runiac.io)

## Install

**homebrew tap**:

```bash
brew install optum/tap/runiac
```

**manually**:

Download the pre-compiled binaries from the [releases](https://github.com/Optum/runiac/releases) page and copy to the desired location.

## Tutorial

For more detailed examples of runiac, be sure to check out the [examples](examples/) directory!

## Using runiac

To use runiac to deploy your infrastructure as code, you will need:

1. `Docker` installed locally
2. `runiac` installed locally

### Inputs

Execute `runiac deploy -h`

## Contributing

Please read [CONTRIBUTING.md](./CONTRIBUTING.md) first.

### Running Locally

runiac is only executed locally with unit tests. To verify changes with the example projects locally, one would need to build the runiac deploy container locally first.

Docker Build:

```bash
$ DOCKER_BUILDKIT=1 docker build -t runiacdeploydev .
```

We recommend adding an alias to install the cli locally:

`alias runiacdev='(cd <LOCAL_PROJECT_LOCATION>/cmd/cli && go build -o $GOPATH/bin/runiacdev) && runiacdev'`

This allows one to use the the `examples` for iterating on runiac changes.

```bash
$ cd examples/...
$ runiacdev -a <YOUR_GCP_PROJECT_ID> -e nonprod --local --container runiacdeploydev
```

> NOTE: If only making changes to the CLI, you do not need to build the container locally `--container runiacdeploydev`
