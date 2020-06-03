# Antidote-Core

[![Build Status](https://travis-ci.org/nre-learning/antidote-core.svg?branch=master)](https://travis-ci.org/nre-learning/antidote-core)
[![codecov](https://codecov.io/gh/nre-learning/antidote-core/branch/master/graph/badge.svg)](https://codecov.io/gh/nre-learning/antidote-core)
[![Go Report Card](https://goreportcard.com/badge/github.com/nre-learning/antidote-core)](https://goreportcard.com/report/github.com/nre-learning/antidote-core)

This repository is where the core services of the Antidote project are maintained. This includes `antidoted`, which is the primary service for provisioning curriculum content within a Kubernetes infrastructure.

If you're looking for the web front-end for Antidote, that code is maintained in a separate repository: [`antidote-web`](https://github.com/nre-learning/antidote-web).

The [Antidote documentation](https://docs.nrelabs.io/antidote/antidote-architecture) contains additional architectural details.

## Hacking



To build `antidote-core`, you'll need Go installed. Any modern version of Go should suffice, but the "officially" supported version is whatever is listed in the Dockerfile. Within the antidote-core repository, compile binaries with:

```
make
```

To run tests:

```
make test
```

If you want to run the core server (`antidoted`) - you'll need to provide a configuration file. Below is a minimal version with only the `api` service enabled (see the `config` package for all supported config options):

```yaml
---
curriculumDir: /path/to/curriculum
instanceId: antidote-dev
enabledServices:
- api
# - stats
# - scheduler
```

You will also need to run a copy of NATS - this docker one-liner should do the trick:

```
docker run --rm -d -p 4222:4222 -p 6222:6222 -p 8222:8222 --name nats-main nats
```

Note that `antictl` and `antidote` are also compiled alongside `antidoted`. They do not have third-party runtime dependencies like `antidoted` does, but may depend on access to a running `antidoted` instance in order to be fully functional (especially `antictl`).
