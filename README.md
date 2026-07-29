# coverlint

[![`Workflow for Taskotter store Action`](https://github.com/gostafa/coverlint/actions/workflows/main.yml/badge.svg)](https://github.com/gostafa/coverlint/actions/workflows/main.yml)
[![codecov](https://codecov.io/gh/gostafa/coverlint/graph/badge.svg)](https://codecov.io/gh/gostafa/coverlint)

`coverlint` checks Go test coverage for each package. The default minimum is **80%**.

It can run as:

* a standalone CLI;
* a plugin inside a custom `golangci-lint` binary.

## Use as a CLI

### Install

Add the module:

```bash
go get github.com/gostafa/coverlint@latest
```

Install the command:

```bash
go install github.com/gostafa/coverlint/cmd/coverlint@latest
```

### Run

```bash
coverlint

# Require 85% coverage.
# coverlint -min 85

# Check selected packages.
# coverlint -min 85 ./internal/...

# Exclude generated packages.
# coverlint -exclude '**/generated/**' ./...

# Open the HTML coverage report.
# coverlint -web
```

Flags must come before package patterns:

```bash
coverlint -min 85 ./...
```

### Build from source

```bash
git clone https://github.com/gostafa/coverlint.git
cd coverlint

go build -o ./bin/coverlint ./cmd/coverlint
./bin/coverlint

# Example:
# ./bin/coverlint -min 85 ./...
```

## Use as a golangci-lint plugin

The plugin must be included in a custom `golangci-lint` binary.

Create `.custom-gcl.yml`:

```yaml
version: v2.12.2
name: custom-golangci-lint
destination: ./bin
plugins:
  - module: github.com/gostafa/coverlint
    version: v0.0.1

    # For local development, replace version with:
    # path: .
```

Enable it in `.golangci.yml`:

```yaml
version: "2"

linters:
  enable:
    - coverlint

  settings:
    custom:
      coverlint:
        type: module
        settings:
          min: 80

          # exclude:
          #   - '**/generated/**'

          # overrides:
          #   - pattern: '**/internal/critical/**'
          #     min: 95

          # packages:
          #   - ./...

          # timeout: 20m

          # test-args:
          #   - -race
```

Build and run the custom linter:

```bash
golangci-lint custom -v
./bin/custom-golangci-lint run ./...
```

Always run the generated `custom-golangci-lint` binary. The standard `golangci-lint` binary does not include the plugin.
