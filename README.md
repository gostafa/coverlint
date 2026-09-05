# coverlint


[![`Workflow for coverlint Action`](https://github.com/gostafa/coverlint/actions/workflows/main.yml/badge.svg)](https://github.com/gostafa/coverlint/actions/workflows/main.yml)
[![codecov](https://codecov.io/gh/gostafa/coverlint/graph/badge.svg)](https://codecov.io/gh/gostafa/coverlint)

`coverlint` checks Go test coverage for each package. Thresholds are fractions
in `[0, 1]` (`0.80` = 80%). Empty rules default to `**` / `0.80`.

It can run as:

* a standalone CLI;
* a plugin inside a custom `golangci-lint` binary.

## Use as a CLI

### Install

```bash
go get github.com/gostafa/coverlint@latest
go install github.com/gostafa/coverlint/cmd/coverlint@latest
```

The public API lives in `github.com/gostafa/coverlint/coverlint`.

### Run

```bash
coverlint

# Check selected packages.
# coverlint ./internal/...

# Require 80% everywhere and 20% under internal.
# coverlint --rule='**':0.80 --rule='**/internal/**':0.2 ./...

# Open the HTML coverage report.
# coverlint --web ./...

# Bound how long `go test` may run, and pass extra test flags.
# coverlint --timeout=20m --test-arg=-race ./...

# Persist go test output and the coverprofile when paths are set.
# coverlint --test-result-path=test-output.txt \
#   --coverage-result-path=coverage.out ./...
```

Flags must come before package patterns:

```bash
coverlint --rule='**':0.80 ./...
```

Useful flags:

* `--rule=pattern:min` (repeatable)
* `--timeout=duration`
* `--test-arg=flag` (repeatable)
* `--test-result-path=path` (omit or leave empty → no file; relative or absolute)
* `--coverage-result-path=path` (omit or leave empty → no file; relative or absolute)
* `--web`
* `--version`

`--test-result-path` writes the combined `go test` stdout/stderr text.
`--coverage-result-path` writes the in-memory coverprofile (`mode: atomic`…).
Relative paths resolve against the process working directory. Parent
directories are created when needed. Writes happen after a successful
toolchain check, including soft-fail lint outcomes, so CI can collect
artifacts when coverage or test-failure diagnostics are reported.

Policy gates package coverage by import-path glob. `min` is a fraction in
`[0, 1]`. Coverage in messages stays a percentage (`coverage 50.00% is below
80.00%`). When multiple rules match, the most specific pattern wins: more
literal segments, then fewer wildcards, then longer patterns; exact ties use
the later rule.

When `go test` fails but a coverprofile is still usable, coverlint continues
coverage evaluation and reports the test failures as lint issues (CLI exit
`1`), instead of aborting as a toolchain/usage error.

### Build from source

```bash
git clone https://github.com/gostafa/coverlint.git
cd coverlint

go build -o ./bin/coverlint ./cmd/coverlint
./bin/coverlint
```

## Use as a golangci-lint plugin

The plugin must be included in a custom `golangci-lint` binary.

Create `.custom-gcl.yml`:

```yaml
version: v2.12.2
name: custom-golangci-lint
destination: ./bin
plugins:
  - module:  github.com/gostafa/coverlint
    import:  github.com/gostafa/coverlint/plugin
    version: v0.0.4
```

Enable it in `.golangci.yml`:

```yaml
version: "2"

linters:
  default: all
  enable:
    - coverlint

  settings:
    custom:
      coverlint:
        type: module
        settings:
          rules:
            - pattern: '**'
              min: 0.80
            - pattern: '**/*_test'
              min: 0.0
          # Optional artifact paths (omit → no file; relative or absolute).
          # test-result-path: test-output.txt
          # coverage-result-path: coverage.out
```

Build and run the custom linter:

```bash
golangci-lint custom -v
./bin/custom-golangci-lint run ./...
```

Always run the generated `custom-golangci-lint` binary. The standard
`golangci-lint` binary does not include the plugin.

Recoverable `go test` failures with a usable coverprofile are reported via
`pass.Report` like coverage threshold misses; they no longer abort the plugin
load as a toolchain error.

## Exit codes

* `0`: success
* `1`: check or write error
* `2`: command usage error
