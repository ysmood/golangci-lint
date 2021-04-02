# Overview

A manager to automate golangci-lint, such as auto-download the executable of golangci-lint.

## Install and run

If the Go version is greater than [v1.17](https://github.com/golang/go/issues/42088):

```bash
go run github.com/ysmood/golangci-lint@latest
```

If the Go version is greater than v1.15:

```bash
go install github.com/ysmood/golangci-lint@latest
golangci-lint
```

If the Go version is less than v1.16:

```bash
go get github.com/ysmood/golangci-lint
golangci-lint
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
