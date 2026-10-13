# Analyzer Integration Exception

The public `New` function may return `*analysis.Analyzer` from
`golang.org/x/tools/go/analysis`. This is the required Go analysis integration
contract. `Settings` remains facade-owned and exposes only facade or standard
library types. The exception does not authorize vendor dependencies in core code.

Owner: analyzer integration maintainers. Risk: the public integration follows the
SDK's compatibility requirements. Remove this exception when the integration is
removed or the SDK supports a portable replacement contract. Architecture tests
enforce the exception's type and package scope.
