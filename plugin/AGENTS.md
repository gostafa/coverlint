# Golangci-Lint Integration Exception

`Plugin` and `BuildAnalyzers` may return `[]*analysis.Analyzer`, and registration
may assert conformance to `register.LinterPlugin`. These SDK contracts are required
by the existing golangci-lint integration. Configuration and coverage results
continue to use facade-owned or standard library types.

Owner: plugin integration maintainers. Risk: registration follows the SDK's
compatibility requirements. Remove this exception when the integration is removed
or the SDK supports portable replacement contracts. Architecture tests enforce
the public signature exception's type and package scope.
