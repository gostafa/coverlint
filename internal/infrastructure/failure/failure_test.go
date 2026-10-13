// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package failure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestTranslationKeepsStableIdentitiesAndHidesCause(t *testing.T) {
	t.Parallel()
	category := errors.New("boundary category")
	private := &os.PathError{Op: "open", Path: "fixture", Err: os.ErrPermission}
	err := Translate(errors.Join(private, context.Canceled), category)
	if !errors.Is(err, category) || !errors.Is(err, context.Canceled) ||
		!errors.Is(err, os.ErrPermission) {
		t.Fatalf("identities lost: %v", err)
	}
	var exposed *os.PathError
	if errors.As(err, &exposed) || errors.Unwrap(err) != nil {
		t.Fatalf("concrete cause escaped: %v", err)
	}
	if err.Error() != errors.Join(private, context.Canceled).Error() {
		t.Fatal("diagnostic changed")
	}
	if errors.Is(err, errors.New("unrelated")) {
		t.Fatal("matched unrelated error")
	}
	if Translate(nil, category) != nil {
		t.Fatal("nil became failure")
	}
	for _, candidate := range []error{context.Canceled, context.DeadlineExceeded, os.ErrPermission} {
		if !errors.Is(Translate(fmt.Errorf("wrapped: %w", candidate), category), candidate) {
			t.Fatalf("lost %v", candidate)
		}
	}
}
