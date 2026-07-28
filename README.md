# coverlint

`coverlint` enforces a minimum Go test coverage percentage for each selected package.

It can run as either:

- a standalone command-line tool;
- a module plugin compiled into a custom `golangci-lint` binary.

The default policy requires **80% coverage per package**. Coverage profiles are created in the system temporary directory, loaded into memory, and removed automatically.

```bash
coverlint
```

## Requirements

- Go 1.26.5 or newer;
- a Go module containing the packages to check;
- `golangci-lint` only when using the plugin integration.

## Quick start

### Standalone CLI

Build the command from this repository:

```bash
go build -o ./bin/coverlint ./cmd/coverlint
```

Check every package under the current module with the default 80% minimum:

```bash
./bin/coverlint
```

Require 85% instead:

```bash
./bin/coverlint -min 85
```

### golangci-lint plugin

Build the custom binary described by `.custom-gcl.yml`:

```bash
golangci-lint custom -v
```

Enable `coverlint` in `.golangci.yml`:

```yaml
version: "2"

linters:
  enable:
    - coverlint
  settings:
    custom:
      coverlint:
        type: module
```

Run the generated binary, not the standard `golangci-lint` executable:

```bash
./custom-golangci-lint run ./...
```

With no `settings` block, the plugin checks `./...` and requires 80% coverage for every package.

## Package selection and policy matching

`coverlint` uses two different kinds of patterns. They serve different purposes and are not interchangeable.

| Pattern type | Examples | Purpose |
|---|---|---|
| Go package pattern | `./...`, `./internal/...` | Selects packages passed to `go test` and `go list`. |
| Import-path glob | `**/internal/**`, `**/generated/**` | Applies an override or exclusion to selected package import paths. |

For example:

```bash
coverlint \
  -min 75 \
  -override '**/internal/**=85' \
  -exclude '**/generated/**' \
  ./...
```

Here, `./...` selects packages. The quoted glob patterns then match complete import paths such as `github.com/acme/project/internal/orders`.

## Coverage policy

### Default minimum

The default minimum is 80%. Set another value with `-min` or the plugin `min` setting.

CLI:

```bash
coverlint -min 85
```

Plugin:

```yaml
settings:
  min: 85
```

Minimum values must be finite, greater than `0`, and at most `100`. Decimal values are supported:

```bash
coverlint -min 87.5
```

A package passes when its calculated coverage is equal to or greater than its required minimum.

### Package overrides

Overrides assign a different minimum to matching package import paths.

CLI:

```bash
coverlint \
  -min 75 \
  -override '**/internal/critical/**=95' \
  -override '**/internal/**=85'
```

Plugin:

```yaml
settings:
  min: 75
  overrides:
    - pattern: '**/internal/critical/**'
      min: 95
    - pattern: '**/internal/**'
      min: 85
```

Overrides are evaluated from top to bottom. **The first matching override wins.** The default `min` value is used when no override matches.

Order specific patterns before broad patterns:

```yaml
overrides:
  - pattern: '**/internal/critical/**'
    min: 95
  - pattern: '**/internal/**'
    min: 85
```

Reversing those entries would cause the broader `**/internal/**` rule to match first.

### Exclusions

Exclusions skip matching package import paths entirely.

CLI:

```bash
coverlint \
  -exclude '**/generated/**' \
  -exclude '**/mocks/**'
```

Plugin:

```yaml
settings:
  exclude:
    - '**/generated/**'
    - '**/mocks/**'
```

Exclusions are evaluated before overrides and the default minimum. Excluded packages count as skipped, not passed or failed.

## Glob pattern reference

Policy globs are matched against the **entire package import path** using `/` as the separator on every operating system.

Given this import path:

```text
github.com/acme/project/internal/orders
```

Supported syntax:

| Syntax | Meaning | Example |
|---|---|---|
| `*` | Zero or more characters within one path segment. It never crosses `/`. | `github.com/acme/*/orders` |
| `?` | Exactly one character within one path segment. | `**/mock?` |
| `[abc]` | One character from a class. Ranges and negated classes follow Go `path.Match` syntax. | `**/[io]rders` |
| `**` | Zero or more complete path segments. It must be a complete segment. | `**/internal/**` |

Common patterns:

| Glob | Matches |
|---|---|
| `**` | Every package import path. |
| `**/internal/**` | Any package named `internal` and all packages below it. |
| `**/generated/**` | Any package named `generated` and all packages below it. |
| `github.com/acme/*/api` | An `api` package with exactly one segment between `acme` and `api`. |
| `github.com/acme/project` | Only that exact import path. |
| `github.com/acme/project/**` | The module root and every package below it. |

Important rules:

- matching is anchored to the complete import path;
- `*` does not cross a `/` separator;
- `**` may cross any number of segments, including zero;
- `**` must appear by itself between separators;
- empty path segments are invalid;
- use `/`, not operating-system file separators;
- quote globs in shell commands to prevent the shell from expanding them before `coverlint` receives them.

Valid:

```text
**/internal/**
github.com/acme/*/api
**/mock?
```

Invalid:

```text
/internal/**
internal//**
internal/**suffix
internal/[
```

These are glob patterns, not regular expressions. For example, use `**/internal/**`, not `^.*/internal/.*$`.

## CLI reference

### Syntax

```text
coverlint [flags] [package-pattern ...]
```

When no package patterns are provided, `./...` is used.

The CLI uses Go's standard `flag` package, so every flag must appear before the first package pattern.

Correct:

```bash
coverlint -min 85 ./internal/...
```

Incorrect:

```bash
coverlint ./internal/... -min 85
```

### Flags

| Flag | Default | Description |
|---|---:|---|
| `-min PERCENT` | `80` | Minimum coverage for selected packages without a matching override. |
| `-override GLOB=MIN` | none | Package-specific minimum. Repeatable; first matching override wins. |
| `-exclude GLOB` | none | Skip matching package import paths. Repeatable. |
| `-test-arg ARG` | none | Pass an additional argument to `go test`. Repeatable. |
| `-timeout DURATION` | `10m` | Maximum duration for coverage collection and package discovery. |
| `-web` | `false` | Open Go's standard HTML coverage report after evaluation. |
| `-version` | — | Print the version and exit. |
| `-h` | — | Print command help and exit. |

### `-min`

Pass the percentage as a number without a `%` suffix:

```bash
coverlint -min 90
coverlint -min 87.5
```

### `-override`

The format is `GLOB=MIN`:

```bash
coverlint -override '**/payments/**=95'
```

A `%` suffix is accepted for an override value:

```bash
coverlint -override '**/payments/**=95%'
```

Repeat the flag for multiple overrides:

```bash
coverlint \
  -override '**/critical/**=95' \
  -override '**/internal/**=85'
```

Because the first match wins, place the most specific override first.

### `-exclude`

```bash
coverlint -exclude '**/generated/**'
```

Repeat the flag for multiple exclusions:

```bash
coverlint \
  -exclude '**/generated/**' \
  -exclude '**/mocks/**'
```

### `-test-arg`

Use one flag for each additional argument passed to `go test`:

```bash
coverlint \
  -test-arg=-race \
  -test-arg=-tags=integration
```

Use the `-test-arg=value` form when the value starts with `-`.

Common examples:

```bash
coverlint -test-arg=-short
coverlint -test-arg=-run=TestUnit
coverlint -test-arg=-tags=integration
coverlint -test-arg=-race -timeout=20m
```

The following arguments are managed by `coverlint` and are rejected:

- `-coverprofile`;
- `--coverprofile`;
- `-covermode`;
- `--covermode`;
- `-count`;
- `--count`.

`-args` and `--` are also rejected because they would stop Go from seeing the package patterns appended by `coverlint`.

`coverlint` always runs tests with `-count=1`, `-covermode=atomic`, and its own temporary `-coverprofile` path.

### `-timeout`

The value uses Go duration syntax:

```bash
coverlint -timeout 30s
coverlint -timeout 20m
coverlint -timeout 1h
```

The timeout covers both `go test` and `go list`. It must be greater than zero.

### `-web`

Open Go's standard HTML coverage report:

```bash
coverlint -web
```

After evaluating the policy, `coverlint` runs the equivalent of:

```text
go tool cover -html=<temporary-profile>
```

Use this option for local development. It may not work in a headless CI environment.

### `-version`

```bash
coverlint -version
```

## CLI examples

### Default policy

```bash
coverlint
```

Checks `./...` with an 80% minimum.

### Selected packages

```bash
coverlint -min 85 ./internal/orders/... ./internal/payments/...
```

### Layered package thresholds

```bash
coverlint \
  -min 75 \
  -override '**/internal/critical/**=95' \
  -override '**/internal/**=85' \
  ./...
```

### Exclude generated packages

```bash
coverlint \
  -min 85 \
  -exclude '**/generated/**' \
  -exclude '**/mocks/**' \
  ./...
```

### Integration tests

```bash
coverlint \
  -min 80 \
  -test-arg=-tags=integration \
  -timeout=20m
```

### Race-enabled tests

```bash
coverlint \
  -min 80 \
  -test-arg=-race \
  -timeout=20m
```

### Local HTML report

```bash
coverlint -min 80 -web
```

### CI script

```bash
set -euo pipefail

go build -o ./bin/coverlint ./cmd/coverlint
./bin/coverlint -min 85 ./...
```

## CLI output and exit codes

Coverage violations are written to standard output. Summaries and execution errors are written to standard error.

Example violation:

```text
internal/payments/service.go:1:1: coverage 72.40% is below 85.00% for package "github.com/acme/project/internal/payments" (181/250 statements) (coverlint)
```

Example success summary:

```text
coverlint: passed (12 checked, 2 skipped)
```

Example failure summary:

```text
coverlint: failed with 2 issue(s) (12 checked, 2 skipped)
```

| Exit code | Meaning |
|---:|---|
| `0` | Every checked package met its required minimum. |
| `1` | At least one package was below its minimum or had no coverage data. |
| `2` | Configuration was invalid, tests failed, the timeout expired, or a required command could not run. |

## golangci-lint module plugin

The plugin is registered under the name `coverlint`. It must be compiled into a custom `golangci-lint` binary.

### Repository module path

This example repository currently uses the placeholder module path:

```text
github.com/gostafa/coverlint
```

Replace it with the real repository path before publishing. Then update imports and run:

```bash
go mod tidy
```

Commit the resulting `go.sum` file.

### Local custom build

The included `.custom-gcl.yml` loads the plugin from the current directory:

```yaml
version: v2.12.2
name: custom-golangci-lint
plugins:
  - module: github.com/gostafa/coverlint
    path: .
```

Build the custom binary:

```bash
golangci-lint custom -v
```

Run the plugin using that generated binary:

```bash
./custom-golangci-lint run -c .golangci.example.yml ./...
```

The standard `golangci-lint` binary does not contain the plugin.

### Published plugin build

After publishing the module, reference a released version in `.custom-gcl.yml`:

```yaml
version: v2.12.2
name: custom-golangci-lint
plugins:
  - module: github.com/your-org/coverlint
    version: v0.5.0
```

The plugin is registered from the module root, so an `import` field is not required.

### Minimal linter configuration

```yaml
version: "2"

linters:
  enable:
    - coverlint
  settings:
    custom:
      coverlint:
        type: module
```

Defaults:

- minimum: `80`;
- packages: `./...`;
- timeout: `10m`;
- coverage mode: `atomic`;
- no overrides;
- no exclusions;
- no additional test arguments.

### Complete linter configuration

```yaml
version: "2"

linters:
  enable:
    - coverlint
  settings:
    custom:
      coverlint:
        type: module
        description: Enforce minimum Go test coverage.
        original-url: github.com/your-org/coverlint
        settings:
          min: 75
          overrides:
            - pattern: '**/internal/critical/**'
              min: 95
            - pattern: '**/internal/**'
              min: 85
          exclude:
            - '**/generated/**'
            - '**/mocks/**'
          packages:
            - ./...
          timeout: 20m
          test-args:
            - -race
            - -tags=integration
```

## Plugin settings reference

Every setting is optional.

| Setting | Type | Default | Description |
|---|---|---|---|
| `min` | number | `80` | Minimum coverage for packages without a matching override. |
| `overrides` | list of objects | empty | Ordered package-specific coverage rules. First match wins. |
| `exclude` | list of strings | empty | Import-path globs for packages to skip. |
| `packages` | list of strings | `./...` | Go package patterns passed to `go test` and `go list`. |
| `timeout` | duration string | `10m` | Maximum duration for the complete check. |
| `test-args` | list of strings | empty | Additional arguments passed to `go test`. |

### `min`

```yaml
settings:
  min: 87.5
```

The value must be finite, greater than `0`, and at most `100`.

### `overrides`

Each override contains:

| Field | Type | Required | Description |
|---|---|---:|---|
| `pattern` | string | yes | Glob matched against the complete package import path. |
| `min` | number | yes | Required coverage percentage for the first matching package. |

```yaml
settings:
  min: 75
  overrides:
    - pattern: '**/critical/**'
      min: 95
    - pattern: '**/internal/**'
      min: 85
```

### `exclude`

```yaml
settings:
  exclude:
    - '**/generated/**'
    - '**/mocks/**'
```

### `packages`

These are Go package patterns, not policy globs:

```yaml
settings:
  packages:
    - ./internal/...
    - ./pkg/...
```

When the standalone Go API supplies package patterns directly, those patterns take precedence over the configured `packages` list. For normal plugin use, configure the list under `settings.packages`.

### `timeout`

```yaml
settings:
  timeout: 20m
```

The value uses Go duration syntax and must be greater than zero.

### `test-args`

```yaml
settings:
  test-args:
    - -race
    - -tags=integration
```

The managed `-coverprofile`, `--coverprofile`, `-covermode`, `--covermode`, `-count`, and `--count` flags are rejected. `-args` and `--` are rejected too.

## How coverage is calculated

For each run, `coverlint`:

1. resolves configuration and validates every glob;
2. runs `go test -count=1 -covermode=atomic` with a temporary coverage profile;
3. runs `go list -json` for the same package patterns and forwards build-context test args such as `-tags`, `-modfile`, `-overlay`, `-race`, `-gcflags`, `-asmflags`, and `-ldflags`;
4. maps coverage blocks to package import paths;
5. merges duplicate profile blocks by file and source range, then totals covered and coverable statements for each package;
6. applies exclusions, then the first matching override, then the default minimum;
7. reports one diagnostic for each package that violates its policy;
8. removes temporary coverage files.

Coverage is enforced **per package**, not as one combined repository-wide percentage.

Only production Go and Cgo source files are used to map packages. Test-only packages, packages whose profile contains no coverable statements, and selected packages with no coverage profile blocks are skipped.

Test failures stop the check before policy evaluation.

## Troubleshooting

### A glob does not match

Check that the pattern targets the complete import path, not a file-system path or a Go package selector.

Use:

```text
**/internal/**
```

Not:

```text
./internal/...
```

The first is a policy glob. The second is a Go package pattern used for package selection.

### The shell changes a glob

Quote every CLI glob:

```bash
coverlint -exclude '**/generated/**'
```

Without quotes, some shells may expand `*`, `?`, or character classes before starting `coverlint`.

### An override is ignored

Overrides use first-match precedence. Move the more specific pattern above the broader pattern.

### A flag is treated as a package

Place all flags before package patterns:

```bash
coverlint -min 85 ./...
```

### Tests fail before coverage is checked

Run the equivalent test command directly and fix the test or build error:

```bash
go test -count=1 -covermode=atomic ./...
```

Include the same values supplied with `-test-arg` or `settings.test-args`.

### The HTML report does not open

`-web` depends on `go tool cover` being able to launch a browser. Avoid it in headless CI environments.

### The plugin is not found

Confirm that:

- the plugin module path in `.custom-gcl.yml` is correct;
- `golangci-lint custom -v` completed successfully;
- the generated `./custom-golangci-lint` binary is the one being executed;
- `coverlint` is enabled under `linters.enable`;
- the custom linter entry uses `type: module`.

## Development

Build the CLI:

```bash
go build -o ./bin/coverlint ./cmd/coverlint
```

Run static checks:

```bash
go vet ./...
```

Run tests for the CLI and internal packages:

```bash
go test ./...
```

Build and run the custom linter:

```bash
make custom-lint
```

Remove generated binaries:

```bash
make clean
```

## License

See [LICENSE](LICENSE).
