package gotool

import (
	"reflect"
	"testing"
)

func TestListArgsForTestArgsForwardsBuildContextOnly(t *testing.T) {
	t.Parallel()

	got := listArgsForTestArgs([]string{
		"-race",
		"-tags=integration",
		"-overlay", "overlay.json",
		"-modfile=tools.mod",
		"-run=TestUnit",
		"-short",
		"-gcflags", "all=-N -l",
	})
	want := []string{
		"-race",
		"-tags=integration",
		"-overlay", "overlay.json",
		"-modfile=tools.mod",
		"-gcflags", "all=-N -l",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listArgsForTestArgs() = %#v, want %#v", got, want)
	}
}

func TestCappedBufferLimitsStoredOutput(t *testing.T) {
	t.Parallel()

	buffer := newCappedBuffer(5)
	if _, err := buffer.Write([]byte("hello world")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := buffer.String(); got != "hello\n... output truncated by coverlint ..." {
		t.Fatalf("String() = %q", got)
	}
}
