// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

const (
	commandOutputLimit            = 1 << 20
	initialPackageCap             = 32
	initialScannerBufferSize      = 64 * 1024
	maxScannerBufferSize          = 4 * 1024 * 1024
	initialProfileBlockCap        = 256
	emptyString                   = ""
	zero                          = 0
	one                           = 1
	two                           = 2
	three                         = 3
	decimalBase                   = 10
	intBitSize                    = 64
	goCommand                     = "go"
	flagPrefix                    = "-"
	positionDot                   = "."
	errWrapLine                   = "%w: %d"
	errWrapJoined                 = "%w: %w"
	errReadCoverageProfileFormat  = "read coverage profile: %w"
	errParseCoverageProfileFormat = "parse coverage profile: %w"
	errEmptyCoverageProfileMsg    = "coverage profile is empty"
	errOpenHTMLFormat             = "open HTML coverage report: %w"
	errReadTempProfileFormat      = "read temporary coverage profile: %w"
	errCloseTempProfileFormat     = "close temporary coverage profile: %w"
	errCloseHTMLInputFormat       = "close temporary HTML coverage input: %w"
	errUnmarshalGoListFormat      = "unmarshal go list package: %w"
	errUnmarshalGoListFieldFormat = "unmarshal %s: %w"
	errCollectCoverageFormat      = "collect coverage: %w"
	errCollectFromProfileFormat   = "collect from profile: %w"
	errCreateTempProfileFormat    = "create temporary coverage profile: %w"
	errWaitGoListDecodeFormat     = "wait go list decode: %w"
	errWaitGoListFormat           = "wait go list: %w"
	errWriteCappedBuffer          = "write capped buffer: %w"
	errDecodeGoListItem           = "decode go list item: %w"
	profileModePrefix             = "mode: "
	truncationSuffix              = "\n... output truncated by coverlint ..."
	flagRace                      = "race"
	flagAsan                      = "asan"
	flagMsan                      = "msan"
	flagTrimpath                  = "trimpath"
	jsonKeyImportPath             = "importPath"
	jsonKeyGoFiles                = "goFiles"
	jsonKeyCgoFiles               = "cgoFiles"
	jsonKeyTestGoFiles            = "testGoFiles"
	jsonKeyDir                    = "dir"
)
