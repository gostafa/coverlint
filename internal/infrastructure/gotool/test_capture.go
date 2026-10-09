// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bytes"
	"encoding/json"
	"slices"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func newTestCapture(output *CappedBuffer) *testCaptureState {
	state := &testCaptureState{
		output:      output,
		logs:        make(map[testKey][]byte),
		packageLogs: make(map[string][]byte),
		reportLogs:  make(map[string][]byte),
		reportOrder: nil,
		passed:      make(map[testKey]bool),
		order:       nil,
		failed:      nil,
		line:        nil,
		drop:        false,
	}

	return state
}

// Write decodes test events and captures their output without returning write errors.
func (capture testCapture) Write(data []byte) (int, error) {
	return capture(data), nil
}

func testCaptureOutput(capture *testCaptureState) testCapture {
	return func(data []byte) int { return writeTestCapture(capture, data) }
}

func writeTestCapture(capture *testCaptureState, data []byte) int {
	for i := range data {
		captureConsume(capture, data[i])
	}

	return len(data)
}

func captureAppendFallbacks(
	capture *testCaptureState,
	result []domain.TestFailure,
) []domain.TestFailure {
	if len(capture.packageLogs[emptyString]) > zero {
		result = append(result, captureFallback(string(capture.packageLogs[emptyString])))
	}

	if capture.output.truncated {
		result = append(result, captureFallback(truncationSuffix))
	}

	return result
}

func captureFallback(output string) domain.TestFailure {
	return domain.TestFailure{ImportPath: emptyString, Test: emptyString, Output: output}
}

func captureConsume(capture *testCaptureState, value byte) {
	if value == '\n' {
		captureFinish(capture)

		return
	}

	if capture.drop {
		return
	}

	if len(capture.line) >= maxScannerBufferSize {
		capture.drop = true
		capture.output.truncated = true

		return
	}

	capture.line = append(capture.line, value)
}

func captureFailedPackages(capture *testCaptureState, fallback []string) []string {
	result := make([]string, zero, len(capture.failed)+len(fallback))

	for i := range capture.failed {
		result = appendUniquePackage(result, capture.failed[i].pkg)
	}

	for i := range fallback {
		result = appendUniquePackage(result, fallback[i])
	}

	return result
}

func captureFailure(capture *testCaptureState, key *testKey) domain.TestFailure {
	output := capture.logs[*key]

	if key.test == emptyString {
		output = capturePackageOutput(capture, key.pkg)
	}

	return domain.TestFailure{ImportPath: key.pkg, Test: key.test, Output: string(output)}
}

func captureFailures(capture *testCaptureState) []domain.TestFailure {
	result := make([]domain.TestFailure, zero, len(capture.failed))

	for i := range capture.failed {
		result = append(result, captureFailure(capture, &capture.failed[i]))
	}

	return captureAppendFallbacks(capture, result)
}

func captureFinish(capture *testCaptureState) {
	if !capture.drop && len(capture.line) > zero {
		captureRecord(capture, capture.line)
	}

	capture.line = capture.line[:zero]
	capture.drop = false
}

// captureGroupOutput preserves event order within each package for text report parsers.
func captureGroupOutput(capture *testCaptureState) {
	output := make([]byte, zero, capture.output.buffer.Len())

	for i := range capture.reportOrder {
		output = append(output, capture.reportLogs[capture.reportOrder[i]]...)
	}

	capture.output.buffer = bytes.NewBuffer(output)
}

func capturePackageOutput(capture *testCaptureState, pkg string) []byte {
	output := slices.Clone(capture.packageLogs[pkg])

	for i := range capture.order {
		key := &capture.order[i]

		if capturePendingTest(capture, key, pkg) {
			output = append(output, capture.logs[*key]...)
		}
	}

	return output
}

func capturePendingTest(capture *testCaptureState, key *testKey, pkg string) bool {
	return key.pkg == pkg && key.test != emptyString && !capture.passed[*key]
}

func captureRecord(capture *testCaptureState, line []byte) {
	var event testEvent

	if json.Unmarshal(line, &event) != nil || event.Action == emptyString {
		key := testKey{pkg: emptyString, test: emptyString}

		captureRecordOutput(capture, &key, string(line)+"\n")

		return
	}

	captureRecordEvent(capture, &event)
}

func captureRecordEvent(capture *testCaptureState, event *testEvent) {
	key := testKey{pkg: event.Package, test: event.Test}

	if event.Action == testActionPass {
		capture.passed[key] = true
	}

	if event.Action == testActionFail {
		capture.failed = append(capture.failed, key)
	}

	if event.Output != emptyString {
		captureRecordOutput(capture, &key, event.Output)
	}
}

func captureRecordOutput(capture *testCaptureState, key *testKey, output string) {
	before := capture.output.buffer.Len()
	writeCappedBytes(
		&cappedWrite{
			buffer:    capture.output.buffer,
			truncated: &capture.output.truncated,
			limit:     capture.output.limit,
		},
		[]byte(output),
	)

	stored := capture.output.buffer.Len() - before

	if stored > zero {
		captureRecordReportOutput(capture, key.pkg, output[:stored])
		captureRecordTestOutput(capture, key, output[:stored])
		captureRecordPackageOutput(capture, key, output[:stored])
	}
}

func captureRecordReportOutput(capture *testCaptureState, pkg, output string) {
	if _, exists := capture.reportLogs[pkg]; !exists {
		capture.reportOrder = append(capture.reportOrder, pkg)
	}

	capture.reportLogs[pkg] = append(capture.reportLogs[pkg], output...)
}

func captureRecordTestOutput(capture *testCaptureState, key *testKey, output string) {
	if _, exists := capture.logs[*key]; !exists {
		capture.order = append(capture.order, *key)
	}

	capture.logs[*key] = append(capture.logs[*key], output...)
}

func captureRecordPackageOutput(capture *testCaptureState, key *testKey, output string) {
	if key.test == emptyString {
		capture.packageLogs[key.pkg] = append(capture.packageLogs[key.pkg], output...)
	}
}

func appendUniquePackage(packages []string, pkg string) []string {
	if pkg == emptyString || slices.Contains(packages, pkg) {
		return packages
	}

	return append(packages, pkg)
}
