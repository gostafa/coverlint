# Repository Agent Instructions

## Architecture

- Use ports and adapters architecture, organized by feature. Keep each feature's domain rules, application use cases, ports, and adapters together under `internal/`; reuse the existing layers when extending a feature.
- Keep domain logic independent of adapters and third-party libraries. Application code depends on domain contracts and ports; adapters implement those ports. Wire concrete adapters at the composition boundary.
- Keep `github.com/gostafa/coverlint/coverlint` as the public facade. Put implementation details in `internal/` and expose only the contracts consumers need.

## Go Package Files

Every Go package may contain only the following Go filenames. Replace `PACKAGENAME` with the actual package name; create only files that have content.

| Filename | Contents |
| --- | --- |
| `funcs.go` | Functions and methods |
| `vars.go` | Package variables, including sentinel errors |
| `consts.go` | Constants |
| `types.go` | Type declarations and package documentation |
| `PACKAGENAME_test.go` | Tests of unexported implementation, using `package PACKAGENAME` |
| `PACKAGENAME_integration_test.go` | Tests of exported behavior, using `package PACKAGENAME_test` and only exported APIs |
| `PACKAGENAME_bench_test.go` | Benchmarks |

Place package documentation on the package declaration in `types.go`. Consolidate Go code into the allowed filenames when changing a package; names such as `doc.go`, `errors.go`, and additional test filenames are outside this convention. Non-Go repository files are unaffected.

## Public API Boundaries

- Keep implementation types, fields, helpers, and state unexported wherever possible. Export symbols inside `internal/` only when another internal package needs them.
- Public signatures, exported fields, type aliases, and interface methods must use public facade types or standard-library types. Keep internal types and third-party adapter types out of the public API.
- Translate implementation errors, events, and options at the facade boundary. Expose public error contracts without leaking concrete internal or third-party errors through error chains.
- Use internal contracts only in private implementation code. Keep consumers independent of internal packages and concrete adapters.

## Required Verification After Every Change

After each change, including documentation or configuration edits, run these commands from the repository root in this order:

```sh
task taskotter:ci:fix
task taskotter:ci
```

Both commands must finish successfully before reporting completion. Inspect changes produced by the fix task. Resolve failures and rerun both commands after further edits until the final tree passes. If an external blocker prevents a successful run, report the exact blocker and failed command; keep the task incomplete rather than claiming success or skipping checks.

## Repository Inspection

Prefer lean-ctx for repository work: `ctx_read` for file reads, `ctx_search` for code searches, and `ctx_shell` or `/opt/homebrew/bin/lean-ctx -c "<command>"` for shell commands. Read `/Users/mostafakhairy/.codex/LEAN-CTX.md` when you need the full routing rules.

## Learned User Preferences

- Fix distance, reusability, and coverlint findings by refactoring, including DTO type aliases and interfaces in stable packages. Do not use `nolint` comments or edit `.golangci.yml` to pass lint checks.
- When asked to make a plan, include at least 10 steps.
- Align folder layout, public interface, and README option coverage with sibling linters `go-reusability` and `go-distance`. Keep README badges, CLI and plugin instructions, and custom golangci-lint documentation consistent with those siblings.
- Register and document the golangci-lint plugin as `github.com/gostafa/coverlint/plugin`, not the module root.

## Learned Workspace Facts

- Coverlint is a Go per-package coverage linter (`github.com/gostafa/coverlint`) that runs as a standalone CLI and as a golangci-lint module plugin.
- The public API package is `github.com/gostafa/coverlint/coverlint`; the plugin package is `github.com/gostafa/coverlint/plugin`.
- Sibling reference implementations are `/Users/mostafakhairy/gostafa/linters/go-distance` and `/Users/mostafakhairy/gostafa/linters/go-reusability`.
- Repository CI requires 100% coverage, including `cmd/coverlint` and the root command. Use the existing testable-main pattern with injectable `run`/`exit` and tests that exercise `main` without `os.Exit`.
- Distance max is 0.30 and reusability min is 0.80; represent DTOs as `type T = struct{...}` aliases so they are not scored as named types.
- Local CI is `task taskotter:ci` / `task taskotter:ci:fix` via Taskotter custom golangci-lint (coverlint, distance, reusability, and vulner).
- Coverage mins are fractions in `[0, 1]`; empty rules default to `**` / `0.80`. The CLI uses repeatable `--rule=pattern:min` (flags before package patterns); the golangci-lint plugin uses `settings.rules` with `pattern` and `min`.
- When multiple rules match, the most specific pattern wins: more literal segments, then fewer wildcards, then longer patterns; exact ties use the later rule.
- Failed `go test` runs are reported as lint issues; coverlint should not hard-fail solely because tests failed.
- There is no `--exclude` / exclude setting on the CLI or plugin; package selection is via rules and package patterns only.
- Optional `--test-result-path` / `--coverage-result-path` (plugin: `test-result-path` / `coverage-result-path`) write those reports only when set; relative paths are accepted.
- Default coverage profile output path is `coverage.out` (not a temp `/tmp/coverlint.coverprofile`).
