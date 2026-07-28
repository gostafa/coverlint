package main

import "testing"

func TestParseOverridesUsesGlobPattern(t *testing.T) {
	t.Parallel()

	overrides, err := parseOverrides([]string{"**/internal/**=85", "**/critical/**=95%"})
	if err != nil {
		t.Fatalf("parseOverrides: %v", err)
	}
	if len(overrides) != 2 {
		t.Fatalf("len(overrides) = %d, want 2", len(overrides))
	}
	if overrides[0].Pattern != "**/internal/**" || overrides[0].Min != 85 {
		t.Fatalf("first override = %#v", overrides[0])
	}
	if overrides[1].Pattern != "**/critical/**" || overrides[1].Min != 95 {
		t.Fatalf("second override = %#v", overrides[1])
	}
}

func TestParseOverridesErrorUsesGlobTerminology(t *testing.T) {
	t.Parallel()

	_, err := parseOverrides([]string{"invalid"})
	if err == nil {
		t.Fatal("parseOverrides succeeded, want error")
	}
	want := `override 1 "invalid" must have the form GLOB=MIN`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}
