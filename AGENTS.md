## Learned User Preferences
- Do not use `nolint` comments to silence golangci-lint findings; refactor the code instead.
- Never edit `.golangci.yml` when remediating lint issues.
- When asked to make a plan, include at least 10 steps.
- Align folder layout, public interface, and README option coverage with sibling linters `go-reusability` and `go-distance`.

## Learned Workspace Facts
- Coverlint is a Go per-package coverage linter (`github.com/gostafa/coverlint`) that runs as a standalone CLI and as a golangci-lint module plugin.
- Sibling reference implementations are `/Users/mostafakhairy/gostafa/linters/go-distance` and `/Users/mostafakhairy/gostafa/linters/go-reusability`.
- Coverage mins are fractions in `[0, 1]`; empty rules default to `**` / `0.80`. The CLI uses repeatable `--rule=pattern:min` (flags before package patterns); the golangci-lint plugin uses `settings.rules` with `pattern` and `min`.
- When multiple rules match, the most specific pattern wins: more literal segments, then fewer wildcards, then longer patterns; exact ties use the later rule.
- Failed `go test` runs are reported as lint issues; coverlint should not hard-fail solely because tests failed.
- There is no `--exclude` / exclude setting on the CLI or plugin; package selection is via rules and package patterns only.
- Optional `--test-result-path` / `--coverage-result-path` (plugin: `test-result-path` / `coverage-result-path`) write those reports only when set; relative paths are accepted.
- Default coverage profile output path is `coverage.out` (not a temp `/tmp/coverlint.coverprofile`).
