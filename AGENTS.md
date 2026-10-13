# Repository Agent Instructions

Applies to the entire repository unless a more specific `AGENTS.md` explicitly overrides a rule for a subtree. MUST / MUST NOT denote constraints; SHOULD / SHOULD NOT denote strong defaults whose deviations need an explicit reason; MAY denotes an optional choice.

## Architecture

Use feature-first ports and adapters (hexagonal) architecture. Preserve feature isolation, core isolation, explicit integration contracts, inward source dependencies, and automated boundary enforcement where practical. Default to the smallest implementation that preserves these invariants. Keep this Go module cohesive; introduce microservices, CQRS, event buses, or additional abstractions only for a concrete requirement. When deployment architecture matters, default to a modular monolith.

### Repository Layout and Responsibilities

Keep each feature's domain rules, application use cases, ports, and adapters together under `internal/features/`. Reuse the existing layers when extending a feature. The following is a reference layout, not a requirement to create empty folders or rename established packages:

```text
coverlint/                         # Public facade and composition boundary
plugin/                            # golangci-lint entry point
cmd/coverlint/                     # Standalone CLI entry point
internal/
├── features/<feature>/
│   ├── domain/                    # Business invariants, decisions, errors
│   ├── application/               # Use cases and workflow orchestration
│   ├── ports/                     # Consumer-owned dependency contracts
│   ├── adapters/<implementation>/ # Input/output translation and external I/O
│   └── public/                    # Published internal feature contracts, if needed
├── cli/                           # CLI delivery and translation
└── infrastructure/                # Reusable technical implementations
```

Co-locate Go tests with their packages using the filenames below. Add domain, port, or published-contract packages only when behavior or boundaries justify them. Shared packages, if needed, own stable feature-neutral concepts; composition roots own construction, configuration, injection, and startup. Neither owns feature business decisions. A feature's `public/` package is an internal cross-feature contract, not the module's consumer API; keep `github.com/gostafa/coverlint/coverlint` as that public facade.

### Feature Boundaries

* **F01 — Feature first.** MUST group implementation by business capability rather than global controller, service, or repository layers.
* **F02 — Self-contained features.** Each feature MUST own its behavior and adapters, except intentionally shared infrastructure.
* **F03 — Cohesive ownership.** A feature SHOULD represent a coherent capability, not an arbitrary technical concern or one data table.
* **F04 — Explicit boundaries.** Features MUST have identifiable private implementation and explicit published contracts when needed.
* **F05 — Inward dependencies.** Source dependencies MUST point toward domain/application contracts. Outward runtime calls through ports do not reverse this direction.
* **F06 — Independent domain.** Domain MUST remain independent of application, adapters, infrastructure, third-party libraries, and other features' private code.
* **F07 — Independent application.** Application MUST depend on domain contracts and ports, without importing or constructing concrete infrastructure, delivery frameworks, database clients, or vendor SDKs.
* **F08 — Private by default.** Features MUST NOT import or manipulate another feature's private implementation. Export only deliberately published contracts.
* **F09 — Controlled integration.** Cross-feature collaboration MUST use published APIs/contracts, a connecting adapter implementing a consumer port, or versioned integration events.
* **F10 — Minimal sharing.** Business rules MUST stay with their owner; shared concepts MUST have demonstrably stable meaning across features.

### Ports

* **F11 — Inbound ports.** Define an explicit inbound interface when useful; an application's callable API MAY itself serve as an inbound port.
* **F12 — Outbound ports.** Application dependencies crossing the core boundary MUST be expressed as interfaces/contracts.
* **F13 — Consumer ownership.** Each adapter MUST implement an explicit port interface owned by the consuming feature. Verify the implementation with a compile-time interface assertion in the adapter package's `vars.go`. Provider-published APIs remain provider-owned.
* **F14 — Capability names.** Ports SHOULD describe capabilities rather than vendors.
* **F15 — Technology-neutral contracts.** Ports MUST use core contract or standard-library types, without framework-specific requests, exceptions, or vendor SDK types.
* **F16 — Interface segregation.** Ports SHOULD expose small, cohesive operations rather than unrelated repository or service capabilities.
* **F17 — No persistence leakage.** Port signatures MUST NOT expose database sessions, query builders, persistence records, or vendor-native payloads.
* **F18 — Explicit semantics.** Document port inputs, outputs, validation, errors, and relevant consistency/idempotency expectations.
* **F19 — Minimal exposure.** Publish only operations callers require; published APIs MUST preserve implementation boundaries.
* **F20 — Necessary interfaces.** Introduce interfaces for actual architectural boundaries or useful substitution; avoid speculative interface-per-type abstractions.

### Adapters

* **F21 — Inbound ownership.** Delivery adapters SHOULD live with the feature they expose; preserve established CLI/plugin entry points and keep them thin.
* **F22 — Outbound ownership.** Outbound adapters SHOULD live with the consuming feature unless intentionally shared as reusable infrastructure.
* **F23 — Input translation.** Inbound adapters MUST translate transport inputs into application commands, queries, or data.
* **F24 — Output translation.** Outbound adapters MUST translate core requests into external operations and map results back to port-defined types.
* **F25 — Boundary validation.** Validate syntax, authentication context, and transport shape at ingress; enforce business invariants in the core.
* **F26 — Core business decisions.** Adapters MUST NOT independently implement feature business policies or workflows.
* **F27 — Stable errors.** Adapters MUST translate vendor failures into stable application-contract errors while retaining diagnostics securely.
* **F28 — Replaceability.** Replacing an adapter SHOULD leave domain/use-case code unchanged when the port contract remains unchanged.
* **F29 — Private adapter isolation.** Adapters MUST NOT directly call another feature's private adapter or repository.
* **F30 — I/O at the edge.** Encapsulate database, network, filesystem, broker, and SDK interactions behind outbound adapters or appropriate boundary services.

### Domain and Application

* **F31 — Rule ownership.** Every business rule MUST have one clear feature owner and authoritative implementation.
* **F32 — Enforced invariants.** Domain entities and value objects MUST protect their relevant invariants and remain focused on domain concepts.
* **F33 — Use-case orchestration.** Application use cases MUST coordinate workflows and calls to ports.
* **F34 — Decisions and orchestration.** Domain objects/services SHOULD decide business rules; application use cases SHOULD orchestrate execution and integration.
* **F35 — Thin boundaries.** Controllers, consumers, repositories, and persistence callbacks MUST NOT become alternative business-logic layers.
* **F36 — Model separation.** Separate domain and persistence representations when coupling would leak persistence or weaken invariants; map at adapter boundaries.
* **F37 — Business failures.** Expose domain/application errors to the core rather than raw transport, SQL, or vendor errors.
* **F38 — Entry-point independence.** Use cases SHOULD support CLI, scheduled jobs, HTTP, or event handlers without rewriting business logic.

### Integration and Infrastructure

* **F39 — Injection.** Supply port implementations from outside the core, using constructor injection or a Go-appropriate equivalent.
* **F40 — Composition root.** Wire concrete adapters at composition boundaries. Roots MAY import private implementations solely for construction and wiring; business logic belongs in features.
* **F41 — Data ownership.** Features MUST access another feature's data through its published contract, never its private tables, schemas, persistence models, or repositories.
* **F42 — Technical reuse.** Reuse technical facilities while keeping core business logic independent of their implementations.
* **F43 — Explicit workflows.** Document cross-feature orchestration in application use cases, published APIs, or events; keep workflow decisions out of adapters.
* **F44 — Consistency.** Define transaction/consistency boundaries deliberately. Prefer atomicity within an owned boundary; use outbox, sagas, idempotency, or eventual consistency across boundaries only as needed. In-process calls do not imply independent transactions.
* **F45 — Resilience.** Configure technical timeouts, backoff, circuit breakers, and external retries in integration code; specify business retry semantics in application code and prevent duplicate effects.

### Testing and Enforcement

* **F46 — Domain tests.** MUST test domain behavior and invariants without real databases, network calls, or application startup.
* **F47 — Use-case tests.** MUST test significant use cases with fakes, spies, or in-memory ports, including important failure paths.
* **F48 — Adapter tests.** SHOULD test adapters against representative real dependencies where appropriate.
* **F49 — Contract tests.** SHOULD verify compatibility of published APIs, ports with multiple implementations, and versioned event schemas.
* **F50 — Architecture checks.** MUST add or maintain language-appropriate import-boundary, forbidden-dependency, and cycle checks where tooling permits, and run them in CI.

Add or update relevant tests when changing important behavior. Select the smallest meaningful set: deterministic core tests for decisions, adapter integration tests for mappings/I/O, contract tests for compatibility, static architecture checks for boundaries, and end-to-end tests for critical workflows. Preserve architectural tests; weakening one requires a documented, authorized architecture change.

### Dependency and Visibility Policy

| From | Allowed dependencies | Forbidden dependencies |
| --- | --- | --- |
| Domain | Own domain, stable shared kernel, standard library | Application, adapters, infrastructure, third-party libraries, other feature internals |
| Application | Own domain, own ports/contracts, stable shared kernel | Concrete adapters, vendor SDKs, delivery frameworks |
| Inbound adapter | Application API and required mapping types | Other feature private implementations |
| Outbound adapter | Consumed port/contracts and external libraries | Other feature private adapters/repositories |
| Feature | Another feature's published API/event contract | Another feature's private domain, adapters, persistence, or tables |
| Composition root | Implementations needed for wiring | Business decisions |
| Shared kernel | Feature-neutral concepts and standard library | Specific features or infrastructure |

Domain MAY define its own interfaces; it MUST NOT import application merely to access a port. An orchestrating application MAY call another feature's published API directly only when it is explicitly accepted as a stable application-level contract without undesirable coupling. Prefer a consumer-owned port and connecting adapter when isolation, substitution, or independent testing matters. MUST NOT introduce dependency cycles; revisit contract ownership, extract a genuinely shared concept, or introduce explicit orchestration when dependencies become mutual.

### Cross-Feature Communication

For synchronous interaction, the consumer defines its needed outbound port, the provider publishes a documented contract, and a consumer-side adapter delegates to that contract. Wire both at the composition root and map types/errors at the boundary. Keep provider-private entities and persistence records behind the provider contract.

When asynchronous interaction is required:

* Distinguish private domain events from published integration events with stable, documented schemas.
* Place transport-specific publication and consumption in adapters. Include event ID, type, version, occurrence time, and relevant correlation/causation metadata.
* Design for the platform's delivery guarantee, including idempotency and safe retries for at-least-once delivery.
* Use a transactional outbox when database commit and event publication need reliable coordination.
* Define compatibility/versioning, ordering, dead-letter handling, and replay behavior where applicable.
* Publish facts through contracts rather than shared mutable objects or hidden state.

Document ownership of shared business facts and how consumers receive updates. Choose an explicit consistency model, avoid distributed transactions where possible, and make retried operations safe for repeated execution.

### Shared Code, Data, and Security

Before sharing code, verify that multiple features need the same semantics, ownership and contract are stable, dependency direction is preserved, and the shared dependency is meaningfully better than duplication. Keep code local until a stable abstraction emerges. Appropriate shared concepts include common identifiers, generic result/pagination primitives, and feature-neutral helpers; feature DTOs used to bypass contracts, mega-repositories, shared persistence entities exposing private data, and arbitrary utility collections are inappropriate.

* Features SHOULD own transaction boundaries, migrations, and persistence mappings when applicable. A shared physical database does not grant cross-feature data access.
* Separate transport DTOs, published DTOs, domain objects, and persistence models when responsibilities differ; avoid redundant mappings.
* Authenticate at delivery boundaries and authorize business actions in application/domain code. Each entry point MUST enforce its required authorization.
* Validate untrusted external inputs at ingress and enforce domain invariants independently.
* Keep credentials, connection details, raw SQL failures, and sensitive implementation details out of published contracts.
* Add logging, tracing, and metrics at boundaries without making instrumentation control business outcomes.
* Make external side effects and failure/rollback behavior explicit.

### Implementation Workflow and Architecture Changes

Before editing, inspect relevant code, entry points, module manifests, tests, and conventions. Identify the owning feature/data and trace delivery, use case, domain decisions, ports, adapters, and published contracts. Check proposed dependencies and preserve established contracts unless the task requires a change.

Implement decisions in domain code, orchestration in application code, translation/I/O in adapters, and wiring at composition boundaries. Keep scope and directories proportional to actual behavior; introduce ports only for real boundaries and use published contracts for cross-feature work. Add tests for behavior, boundary failures, and contract changes.

Before finishing, run the required checks below, inspect changed imports/boundaries for forbidden dependencies or cycles, and confirm ownership, errors, consistency, authorization, and side effects. Summarize what changed, its owning feature, validation, and material limitations or deliberate tradeoffs. Report checks that could not run precisely.

If a requirement conflicts with these boundaries, describe the rule and requirement, prefer a solution preserving the five invariants, and document any unavoidable exception's scope, owner, rationale, risks, and removal plan in the existing architecture-decision mechanism. Follow any explicit repository architecture-review process; preserve contracts, tests, visibility, and ownership unless the change explicitly authorizes revising them.

A change is complete only when it implements the request, respects feature ownership and inward dependencies, provides adequate tests for changed behavior, passes mandated verification, and introduces no unauthorized boundary violation.

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

* Keep implementation types, fields, helpers, and state unexported wherever possible. Export symbols inside `internal/` only when another internal package needs them.
* Public signatures, exported fields, type aliases, and interface methods must use public facade types or standard-library types. Keep internal types and third-party adapter types out of the public API.
* Translate implementation errors, events, and options at the facade boundary. Expose public error contracts without leaking concrete internal or third-party errors through error chains.
* Use internal contracts only in private implementation code. Keep consumers independent of internal packages and concrete adapters.

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

* Fix distance, reusability, and coverlint findings by refactoring, including DTO type aliases and interfaces in stable packages. Do not use `nolint` comments or edit `.golangci.yml` to pass lint checks.
* When asked to make a plan, include at least 10 steps.
* Align folder layout, public interface, and README option coverage with sibling linters `go-reusability` and `go-distance`. Keep README badges, CLI and plugin instructions, and custom golangci-lint documentation consistent with those siblings.
* Register and document the golangci-lint plugin as `github.com/gostafa/coverlint/plugin`, not the module root.

## Learned Workspace Facts

* Coverlint is a Go per-package coverage linter (`github.com/gostafa/coverlint`) that runs as a standalone CLI and as a golangci-lint module plugin.
* The public API package is `github.com/gostafa/coverlint/coverlint`; the plugin package is `github.com/gostafa/coverlint/plugin`.
* Sibling reference implementations are `/Users/mostafakhairy/gostafa/linters/go-distance` and `/Users/mostafakhairy/gostafa/linters/go-reusability`.
* Repository CI requires 100% coverage, including `cmd/coverlint` and the root command. Use the existing testable-main pattern with injectable `run`/`exit` and tests that exercise `main` without `os.Exit`.
* Distance max is 0.30 and reusability min is 0.80; represent DTOs as `type T = struct{...}` aliases so they are not scored as named types.
* Local CI is `task taskotter:ci` / `task taskotter:ci:fix` via Taskotter custom golangci-lint (coverlint, distance, reusability, and vulner).
* Coverage mins are fractions in `[0, 1]`; empty rules default to `**` / `0.80`. The CLI uses repeatable `--rule=pattern:min` (flags before package patterns); the golangci-lint plugin uses `settings.rules` with `pattern` and `min`.
* When multiple rules match, the most specific pattern wins: more literal segments, then fewer wildcards, then longer patterns; exact ties use the later rule.
* Failed `go test` runs are reported as lint issues; coverlint should not hard-fail solely because tests failed.
* There is no `--exclude` / exclude setting on the CLI or plugin; package selection is via rules and package patterns only.
* Optional `--test-result-path` / `--coverage-result-path` (plugin: `test-result-path` / `coverage-result-path`) write those reports only when set; relative paths are accepted.
* Default coverage profile output path is `coverage.out` (not a temp `/tmp/coverlint.coverprofile`).
