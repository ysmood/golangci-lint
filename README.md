# Overview

A manager to automate golangci-lint, such as auto-download the executable of golangci-lint.

## Install the manager

If the Go version is greater than v1.15:

```bash
go install github.com/ysmood/golangci-lint@latest
```

If the Go version is less than v1.16:

```bash
go get github.com/ysmood/golangci-lint
```

## Arguments

Pass arguments to the manager:

```bash
golangci-lint -h
```

Arguments after the `--` will only be passed to the golangci-lint:

```bash
golangci-lint -- -h
```
