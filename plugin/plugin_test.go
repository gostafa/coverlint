// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package plugin

import (
	"strings"
	"testing"
)

func TestDecodePluginSettingsNil(t *testing.T) {
	t.Parallel()

	settings, err := decodePluginSettings(nil)
	if err != nil {
		t.Fatalf("decodePluginSettings(nil): %v", err)
	}

	if settings.Rules != nil || settings.Packages != nil ||
		settings.Timeout != "" || settings.TestArgs != nil {
		t.Fatalf("settings = %#v, want zero value", settings)
	}
}

func TestDecodePluginSettingsMarshalFailure(t *testing.T) {
	t.Parallel()

	_, err := decodePluginSettings(make(chan int))

	if err == nil || !strings.Contains(err.Error(), "marshal settings") {
		t.Fatalf("error = %v, want marshal settings", err)
	}
}
