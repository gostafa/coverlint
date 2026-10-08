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
	failures := capture.failures()
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
	if failures := capture.failures(); len(failures) != 4 || failures[1].Test != "TestLater" {
		t.Fatalf("failures = %#v", failures)
	}
	if packages := capture.failedPackages(nil); len(packages) != 2 || packages[1] != "b" {
		t.Fatalf("packages = %#v", packages)
	}
	if !strings.Contains(buffer.String(), "output truncated") {
		t.Fatalf("output = %q", buffer.String())
	}
}

func TestCaptureHandlesRawAndOversizedLines(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	_, err := capture.Write(
		[]byte("compiler error\n" + strings.Repeat("x", maxScannerBufferSize+1) + "\n"),
	)
	if err != nil {
		t.Fatal(err)
	}
	writeCaptureEvents(t, capture, []testEvent{{Action: "fail", Package: "a", Test: "TestAfter"}})
	if !strings.Contains(buffer.String(), "compiler error") ||
		capture.failures()[0].Test != "TestAfter" {
		t.Fatalf("capture = %#v, output = %q", capture.failures(), buffer.String())
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
	output := capture.failures()[0].Output
	if strings.Contains(output, "passing log") || !strings.Contains(output, "panic: broken") {
		t.Fatalf("package failure = %q", output)
	}
}

func writeCaptureEvents(t *testing.T, capture *testCapture, events []testEvent) {
	t.Helper()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		// Split the line to exercise arbitrary command-pipe write boundaries.
		for _, chunk := range [][]byte{data[:len(data)/2], append(data[len(data)/2:], '\n')} {
			if _, err := capture.Write(chunk); err != nil {
				t.Fatal(err)
			}
		}
	}
	capture.finish()
}
