# Overview

A lib to automate golangci-lint, such as auto-download the executable of golangci-lint.

Run the command below to lint a project:

```bash
go run github.com/ysmood/golangci-lint
```

Arguments after the `--` will be passed to golangci-lint:

```bash
go run github.com/ysmood/golangci-lint -- run --help
```
