package gotool_test

import (
	"reflect"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/adapters/gotool"
)

func TestListArgsForTestArgsForwardsBuildContextOnly(t *testing.T) {
	t.Parallel()

	got := gotool.ListArgsForTestArgs([]string{
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

	buffer := gotool.NewCappedBuffer(5)

	_, err := buffer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := buffer.String(); got != "hello\n... output truncated by coverlint ..." {
		t.Fatalf("String() = %q", got)
	}
}
