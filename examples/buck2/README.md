# Skaffold Buck2 Example

This example shows how to use [Buck2](https://buck2.build/) as a builder with Skaffold.

## Prerequisites

- [Buck2](https://buck2.build/docs/getting_started/) installed
- Docker running (used by the genrule to build the container image)
- A Kubernetes cluster (e.g. minikube)

## Setup

Initialize the Buck2 prelude in this directory:

```sh
buck2 init --git
```

This creates the `prelude/` directory with Buck2's standard rules.

## Running

```sh
skaffold dev
```

Or to just build:

```sh
skaffold build
```

## How it works

The `BUCK` file defines a `genrule` that:
1. Copies the Go source and Dockerfile to a temp directory
2. Runs `docker build` to create the container image
3. Runs `docker save` to export the image as a `.tar` file

Skaffold then loads this `.tar` into the local Docker daemon (or pushes it to a registry).

## Adapting to your project

For production use, you would typically replace the `genrule` with proper Buck2 rules
for building Go binaries and container images (e.g. using `go_binary` and OCI image rules
from your Buck2 prelude or third-party rules).
