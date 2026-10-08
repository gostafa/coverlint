// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"encoding/json"
	"slices"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func newTestCapture(output *CappedBuffer) *testCapture {
	return &testCapture{
		output:      output,
		logs:        make(map[testKey][]byte),
		packageLogs: make(map[string][]byte),
		passed:      make(map[testKey]bool),
		order:       nil,
		failed:      nil,
		line:        nil,
		drop:        false,
	}
}

func (capture *testCapture) Write(data []byte) (int, error) {
	for i := range data {
		capture.consume(data[i])
	}

	return len(data), nil
}

func (capture *testCapture) appendFallbacks(result []domain.TestFailure) []domain.TestFailure {
	if len(capture.packageLogs[emptyString]) > zero {
		result = append(result, domain.TestFailure{
			ImportPath: emptyString,
			Test:       emptyString,
			Output:     string(capture.packageLogs[emptyString]),
		})
	}

	if capture.output.truncated {
		result = append(
			result,
			domain.TestFailure{
				ImportPath: emptyString,
				Test:       emptyString,
				Output:     truncationSuffix,
			},
		)
	}

	return result
}

func (capture *testCapture) consume(value byte) {
	if value == '\n' {
		capture.finish()

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

func (capture *testCapture) failedPackages(fallback []string) []string {
	result := make([]string, zero, len(capture.failed)+len(fallback))

	for i := range capture.failed {
		result = appendUniquePackage(result, capture.failed[i].pkg)
	}

	for i := range fallback {
		result = appendUniquePackage(result, fallback[i])
	}

	return result
}

func (capture *testCapture) failure(key *testKey) domain.TestFailure {
	output := capture.logs[*key]

	if key.test == emptyString {
		output = capture.packageOutput(key.pkg)
	}

	return domain.TestFailure{ImportPath: key.pkg, Test: key.test, Output: string(output)}
}

func (capture *testCapture) failures() []domain.TestFailure {
	result := make([]domain.TestFailure, zero, len(capture.failed))

	for i := range capture.failed {
		result = append(result, capture.failure(&capture.failed[i]))
	}

	return capture.appendFallbacks(result)
}

func (capture *testCapture) finish() {
	if !capture.drop && len(capture.line) > zero {
		capture.record(capture.line)
	}

	capture.line = capture.line[:zero]
	capture.drop = false
}

func (capture *testCapture) packageOutput(pkg string) []byte {
	output := slices.Clone(capture.packageLogs[pkg])

	for i := range capture.order {
		key := &capture.order[i]

		if capture.pendingTest(key, pkg) {
			output = append(output, capture.logs[*key]...)
		}
	}

	return output
}

func (capture *testCapture) pendingTest(key *testKey, pkg string) bool {
	return key.pkg == pkg && key.test != emptyString && !capture.passed[*key]
}

func (capture *testCapture) record(line []byte) {
	var event testEvent

	if json.Unmarshal(line, &event) != nil || event.Action == emptyString {
		key := testKey{pkg: emptyString, test: emptyString}

		capture.recordOutput(&key, string(line)+"\n")

		return
	}

	capture.recordEvent(&event)
}

func (capture *testCapture) recordEvent(event *testEvent) {
	key := testKey{pkg: event.Package, test: event.Test}

	if event.Action == "pass" {
		capture.passed[key] = true
	}

	if event.Action == "fail" {
		capture.failed = append(capture.failed, key)
	}

	if event.Output != emptyString {
		capture.recordOutput(&key, event.Output)
	}
}

func (capture *testCapture) recordOutput(key *testKey, output string) {
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
		capture.recordTestOutput(key, output[:stored])
		capture.recordPackageOutput(key, output[:stored])
	}
}

func (capture *testCapture) recordTestOutput(key *testKey, output string) {
	if _, exists := capture.logs[*key]; !exists {
		capture.order = append(capture.order, *key)
	}

	capture.logs[*key] = append(capture.logs[*key], output...)
}

func (capture *testCapture) recordPackageOutput(key *testKey, output string) {
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
