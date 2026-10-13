# AGENTS.md Compliance Checklist

This checklist tracks the repository-wide refactor against the repository rules.
The repository uses in-process synchronous integration and filesystem artifacts.
It has no database, HTTP server, authenticated user actions, broker, distributed
transactions, or published asynchronous events. Requirements specific to those
technologies (authentication, migrations, event schemas, outbox, sagas, dead-letter
handling, and distributed tracing) are inapplicable to the current application.

## Architecture Rules

| Rule | Implementation evidence |
| --- | --- |
| F01 | Collection, policy, and reporting own their implementation and delivery adapters. |
| F02 | Adapters live with their consumers; infrastructure contains generic technical facilities. |
| F03 | Features represent collecting inputs, applying policy, and reporting outcomes. |
| F04 | Collection/reporting publish explicit contracts; private implementations stay internal. |
| F05 | Domain/application imports point inward; architecture tests reject reverse imports. |
| F06 | Domains use their own data and standard-library computation, with explicit working-directory inputs. |
| F07 | Applications depend on owned domain and port contracts. |
| F08 | Cross-feature imports target published contracts. |
| F09 | The policy connecting adapter consumes collection’s published API. |
| F10 | Coverage decisions stay in policy; failure merging stays in collection. |
| F11 | Callable use cases serve as inbound APIs. |
| F12 | Collection, imported-report, artifact, and HTML dependencies use ports. |
| F13 | Adapter packages declare compile-time conformance in vars.go. |
| F14 | Ports describe coverage, package discovery, report reading, artifacts, and HTML reporting. |
| F15 | Port data uses owned core or standard-library types. |
| F16 | Policy consumes a cohesive input port and separate read/write ports. |
| F17 | No database, broker, or persistence-record contracts exist. |
| F18 | Port documentation defines errors, cancellation, failure-data handling, and repeatability. |
| F19 | Consumer facades publish owned data; implementation helpers remain private where possible. |
| F20 | Interfaces represent actual I/O and integration boundaries. |
| F21 | CLI/analyzer implementations belong to policy; executable/public paths remain stable. |
| F22 | Collection, artifact, cache, and HTML adapters belong to their consumers. |
| F23 | CLI/settings/facade translate inputs into validated core requests. |
| F24 | Connecting adapters translate owned request/result data and errors. |
| F25 | Ingress parses syntax; policy domain owns minima, rule compilation, managed flags, and defaults. |
| F26 | Adapters delegate policy decisions and failure merging to the core. |
| F27 | Adapters classify failures with port-owned categories; opaque translation hides concrete causes. |
| F28 | Use cases depend on replaceable contracts, tested with fakes. |
| F29 | Adapters reach other features through published APIs. |
| F30 | Process, file, and environment operations occur at boundaries/infrastructure. |
| F31 | Defaults and validation are authoritative in policy; failure merging is authoritative in collection. |
| F32 | Compiled policy validates and copies mutable rules before evaluation. |
| F33 | Policy orchestrates report import, collection, evaluation, and ordered writes. |
| F34 | Domains decide policy; application coordinates ports. |
| F35 | Consumer facades translate and compose; delivery adapters expose use cases. |
| F36 | There is no persistence model; explicit provider/consumer mappings preserve isolation. |
| F37 | Application failures use owned contracts and stable categories. |
| F38 | The same coverage use case supports the facade, CLI, and analyzer. |
| F39 | Composition supplies collector/read/write/reporting dependencies. |
| F40 | Public/executable constructors wire concrete implementations. |
| F41 | Features access provider data through published collection/reporting contracts. |
| F42 | Generic caching, retries, process startup, cleanup, and opaque errors are reused. |
| F43 | Use-case code and contract documentation describe the synchronous workflow. |
| F44 | Artifact writes intentionally remain ordered and nontransactional; partial success is documented. |
| F45 | Existing safe-operation retry bounds remain; started processes and close failures are not retried. |
| F46 | Domain tests run without real external dependencies. |
| F47 | Use-case fake-port tests cover import ordering, output writes, and failure stops. |
| F48 | Existing real Go-tool and filesystem tests remain, with HTML tests moved to reporting. |
| F49 | Contract tests verify mappings, copies, public methods, error identities, and SDK compatibility. |
| F50 | CI runs architecture tests; compiler/import checks reject cycles and invalid dependencies. |

## Additional Rules

* Go declarations use `funcs.go`, `types.go`, `consts.go`, and `vars.go`;
  tests use the prescribed package filenames. Every package includes `doc.go`
  for package documentation. Architecture tests verify its presence, documentation
  placement, filenames, and declaration locations.
* Public coverage types are facade-owned, including nested rule fields.
  Integration signatures retain only the explicitly authorized SDK exception
  documented in analyzer/plugin scoped instructions. Public errors hide internal
  concrete causes while preserving categories and cancellation identities.
* README describes the actual APIs, feature ownership, retry behavior, and
  ordered artifact writes. Synchronized TaskOtter modules and lint thresholds
  remain unchanged; no suppression comments are introduced.
* Scoped instructions inherit the root rules; the SDK exceptions record scope,
  owner, rationale, risk, and removal conditions.

## Verification

Required order: `task taskotter:ci:fix`, inspect automatic changes, then
`task taskotter:ci`. Also run `task architecture:check`, `go test -race ./...`,
`go build ./...`, and `task taskotter:go-junit-report:report`.

The configured coverage threshold remains 100%. The implementation passes the
configured fix/CI sequence, architecture checks, race tests, Go build, and JUnit
report generation. The custom golangci-lint binary also builds from the refactored
local plugin. Repository tooling modules, dependencies, lint configuration, and
thresholds remain unchanged.
