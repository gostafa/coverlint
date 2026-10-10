// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

// ReadTestReport reads optional existing Go text or JSON test results.
func ReadTestReport(path string) (domain.Coverage, error) {
	if path == emptyString {
		return domain.Coverage{}, nil
	}

	file, err := os.Open(filepath.Clean(path))

	if errors.Is(err, os.ErrNotExist) {
		return domain.Coverage{}, nil
	}

	if err != nil {
		return domain.Coverage{}, fmt.Errorf("open test report %q: %w", path, err)
	}

	coverage, parseErr := parseTestReport(bufio.NewScanner(file), path)
	closeErr := file.Close()

	return coverage, errors.Join(parseErr, closeErr)
}

func parseTestReport(scanner *bufio.Scanner, path string) (domain.Coverage, error) {
	parser := newTestReportParser()

	scanner.Buffer(nil, maxScannerBufferSize)

	for scanner.Scan() {
		parseReportLine(parser, scanner.Text())
	}

	flushTextReport(&parser.text, &parser.result, emptyString)
	mergeJSONReport(&parser.result, parser.capture)
	labelReportFailures(&parser.result, path)

	return parser.result, errors.Join(reportScanError(scanner.Err(), path))
}

func newTestReportParser() *testReportParser {
	output := NewCappedBuffer(commandOutputLimit)

	return &testReportParser{capture: newTestCapture(&output)}
}

func parseReportLine(parser *testReportParser, line string) {
	var event testEvent

	if json.Unmarshal([]byte(line), &event) == nil && event.Action != emptyString {
		captureRecordEvent(parser.capture, &event)

		return
	}

	parseTextReportLine(&parser.text, &parser.result, line)
}

func mergeJSONReport(result *domain.Coverage, capture *testCaptureState) {
	if len(capture.failed) == zero {
		return
	}

	imported := domain.Coverage{
		TestsFailed:    true,
		Failures:       captureFailures(capture),
		FailedPackages: captureFailedPackages(capture, nil),
	}
	domain.MergeTestFailures(result, &imported)
}

func labelReportFailures(result *domain.Coverage, path string) {
	for i := range result.Failures {
		result.Failures[i].Output = fmt.Sprintf(
			"existing test report %q:\n%s", path, result.Failures[i].Output,
		)
	}
}

func reportScanError(err error, path string) error {
	if err != nil {
		return fmt.Errorf("parse test report %q: %w", path, err)
	}

	return nil
}

func parseTextReportLine(text *textTestReport, result *domain.Coverage, line string) {
	trimmed := strings.TrimSpace(line)
	appendTextReportOutput(text, line)

	if strings.HasPrefix(trimmed, textTestFailPrefix) {
		text.failed = true
		text.failures = append(text.failures, domain.TestFailure{Test: textFailureName(trimmed)})
	}

	appendTextFailureOutput(text, line)
	finishTextReportLine(text, result, line)
}

func appendTextReportOutput(text *textTestReport, line string) {
	if text.output == nil {
		output := NewCappedBuffer(commandOutputLimit)

		text.output = &output
	}

	writeCappedBytes(&cappedWrite{
		buffer: text.output.buffer, truncated: &text.output.truncated, limit: text.output.limit,
	}, []byte(line+"\n"))
}

func textFailureName(line string) string {
	fields := strings.Fields(strings.TrimPrefix(line, textTestFailPrefix))

	if len(fields) == zero {
		return emptyString
	}

	return fields[zero]
}

func appendTextFailureOutput(text *textTestReport, line string) {
	if len(text.failures) == zero || text.output.truncated {
		return
	}

	if !indentedReportLine(line) {
		return
	}

	text.failures[len(text.failures)-one].Output += line + "\n"
}

func finishTextReportLine(text *textTestReport, result *domain.Coverage, line string) {
	if indentedReportLine(line) {
		return
	}

	fields := strings.Fields(line)

	if len(fields) == zero {
		return
	}

	finishTextReportFields(text, result, fields)
}

func indentedReportLine(line string) bool {
	return strings.HasPrefix(line, spaceSeparator) || strings.HasPrefix(line, tabSeparator)
}

func finishTextReportFields(text *textTestReport, result *domain.Coverage, fields []string) {
	switch fields[zero] {
	case textFailAction:
		finishFailedTextReport(text, result, fields)
	case "ok", "?":
		flushTextReport(text, result, emptyString)
	default:
	}
}

func finishFailedTextReport(text *textTestReport, result *domain.Coverage, fields []string) {
	text.failed = text.failed || !result.TestsFailed || len(fields) > one

	if len(fields) > one {
		flushTextReport(text, result, fields[one])
	}
}

func flushTextReport(text *textTestReport, result *domain.Coverage, pkg string) {
	if text.failed {
		appendTextReportFailures(text, result, pkg)
	}

	*text = textTestReport{}
}

func appendTextReportFailures(text *textTestReport, result *domain.Coverage, pkg string) {
	result.TestsFailed = true
	result.FailedPackages = appendUniquePackage(result.FailedPackages, pkg)

	if len(text.failures) == zero {
		text.failures = append(text.failures, domain.TestFailure{Output: text.output.String()})
	}

	for i := range text.failures {
		text.failures[i].ImportPath = pkg

		if text.output.truncated && text.failures[i].Test != emptyString {
			text.failures[i].Output += truncationSuffix
		}

		result.Failures = append(result.Failures, text.failures[i])
	}
}
