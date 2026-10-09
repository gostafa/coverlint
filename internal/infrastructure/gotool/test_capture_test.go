// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCaptureAttributesParallelFailures(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	events := []testEvent{
		{
			Action:  "output",
			Package: "a",
			Test:    "TestCalc/case",
			Output:  "input: 2\ngot: 3\nwant: 4\n",
		},
		{Action: "output", Package: "b", Test: "TestPass", Output: "passing log\n"},
		{Action: "fail", Package: "a", Test: "TestCalc/case"},
		{Action: "fail", Package: "b", Test: "TestSilent"},
		{Action: "fail", Package: "a"},
	}
	writeCaptureEvents(t, capture, events)
	failures := captureFailures(capture)
	if len(failures) != 3 || failures[0].Test != "TestCalc/case" ||
		failures[1].Test != "TestSilent" {
		t.Fatalf("failures = %#v", failures)
	}
	if failures[0].Output != events[0].Output || failures[1].Output != "" {
		t.Fatalf("failure output = %#v", failures)
	}
	if !strings.Contains(buffer.String(), "passing log") ||
		strings.Contains(buffer.String(), `"Action"`) {
		t.Fatalf("text report = %q", buffer.String())
	}
}

func TestCaptureGroupsInterleavedPackageOutput(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	writeCaptureEvents(t, capture, []testEvent{
		{Action: "output", Package: "a", Output: "=== RUN   TestA\n"},
		{Action: "output", Package: "b", Output: "=== RUN   TestB\n"},
		{Action: "output", Package: "b", Output: "--- PASS: TestB (0.00s)\nok  b\n"},
		{Action: "output", Package: "a", Output: "--- PASS: TestA (0.00s)\nok  a\n"},
	})
	captureGroupOutput(capture)
	want := "=== RUN   TestA\n--- PASS: TestA (0.00s)\nok  a\n" +
		"=== RUN   TestB\n--- PASS: TestB (0.00s)\nok  b\n"
	if buffer.String() != want {
		t.Fatalf("report = %q, want %q", buffer.String(), want)
	}
}

func TestCaptureRetainsNamesAfterTruncation(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(5)
	capture := newTestCapture(&buffer)
	writeCaptureEvents(t, capture, []testEvent{
		{Action: "output", Package: "a", Test: "TestFirst", Output: "long output\n"},
		{Action: "fail", Package: "a", Test: "TestFirst"},
		{Action: "fail", Package: "b", Test: "TestLater"},
		{Action: "fail", Package: "b"},
	})
	if failures := captureFailures(capture); len(failures) != 4 || failures[1].Test != "TestLater" {
		t.Fatalf("failures = %#v", failures)
	}
	if packages := captureFailedPackages(capture, nil); len(packages) != 2 || packages[1] != "b" {
		t.Fatalf("packages = %#v", packages)
	}
	captureGroupOutput(capture)
	if !strings.Contains(buffer.String(), "output truncated") {
		t.Fatalf("output = %q", buffer.String())
	}
}

func TestCaptureHandlesRawAndOversizedLines(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	_, err := testCaptureOutput(capture).Write(
		[]byte("compiler error\n" + strings.Repeat("x", maxScannerBufferSize+2) + "\n"),
	)
	if err != nil {
		t.Fatal(err)
	}
	writeCaptureEvents(t, capture, []testEvent{{Action: "fail", Package: "a", Test: "TestAfter"}})
	captureGroupOutput(capture)
	if !strings.Contains(buffer.String(), "compiler error") ||
		captureFailures(capture)[0].Test != "TestAfter" {
		t.Fatalf("capture = %#v, output = %q", captureFailures(capture), buffer.String())
	}
}

func TestCapturePackageFailureOmitsPassingLogs(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	writeCaptureEvents(t, capture, []testEvent{
		{Action: "output", Package: "a", Test: "TestPass", Output: "passing log\n"},
		{Action: "pass", Package: "a", Test: "TestPass"},
		{Action: "output", Package: "a", Test: "TestPanic", Output: "panic: broken\n"},
		{Action: "output", Package: "a", Output: "FAIL\n"},
		{Action: "fail", Package: "a"},
	})
	output := captureFailures(capture)[0].Output
	if strings.Contains(output, "passing log") || !strings.Contains(output, "panic: broken") {
		t.Fatalf("package failure = %q", output)
	}
}

func writeCaptureEvents(t *testing.T, capture *testCaptureState, events []testEvent) {
	t.Helper()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		// Split the line to exercise arbitrary command-pipe write boundaries.
		for _, chunk := range [][]byte{data[:len(data)/2], append(data[len(data)/2:], '\n')} {
			if _, err := testCaptureOutput(capture).Write(chunk); err != nil {
				t.Fatal(err)
			}
		}
	}
	captureFinish(capture)
}
