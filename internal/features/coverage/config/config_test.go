// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

var errBoom = errors.New("boom")

func TestDecoderFuncDecode(t *testing.T) {
	t.Parallel()

	ok := decoderFunc(func([]byte) error { return nil })
	if err := ok.decode([]byte(`{}`)); err != nil {
		t.Fatalf("decode success: %v", err)
	}

	fail := decoderFunc(func([]byte) error { return errBoom })

	err := fail.decode([]byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "decode coverage config: boom") {
		t.Fatalf("error = %v, want wrapped boom", err)
	}
}

func TestApplyDecodedConfigRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	var cfg Config

	err := applyDecodedConfig(&cfg, []byte(`{"rules":`))
	if err == nil || !strings.Contains(err.Error(), "apply coverage config:") {
		t.Fatalf("error = %v, want apply coverage config error", err)
	}
}

func TestDecodeConfigAliasRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := decodeConfigAlias([]byte(`not-json`))
	if err == nil || !strings.Contains(err.Error(), "decode coverage config fields:") {
		t.Fatalf("error = %v, want decode fields error", err)
	}
}

func TestDecodeRawConfigRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := decodeRawConfig([]byte(`[`))
	if err == nil || !strings.Contains(err.Error(), "decode coverage config:") {
		t.Fatalf("error = %v, want decode coverage config error", err)
	}
}

func TestFinishUnmarshalErrorArms(t *testing.T) {
	t.Parallel()

	var cfg Config

	err := finishUnmarshal(&cfg, []byte(`not-json`))
	if err == nil || !strings.Contains(err.Error(), "unmarshal coverage config:") {
		t.Fatalf("error = %v, want unmarshal error from raw decode", err)
	}

	err = finishUnmarshal(&cfg, []byte(`{"rules":"bad"}`))
	if err == nil || !strings.Contains(err.Error(), "unmarshal coverage config:") {
		t.Fatalf("error = %v, want unmarshal error from apply", err)
	}
}

func TestMarshalRawConfigRejectsMarshalFailure(t *testing.T) {
	_, err := marshalRawConfigWith(
		map[string]json.RawMessage{timeoutKey: json.RawMessage(`"1s"`)},
		func(any) ([]byte, error) { return nil, errBoom },
	)
	if err == nil || !strings.Contains(err.Error(), "encode remapped coverage config: boom") {
		t.Fatalf("error = %v, want marshal failure", err)
	}
}

func TestRemapConfigAliasesErrorArms(t *testing.T) {
	_, err := remapConfigAliases([]byte(`{`))
	if err == nil || !strings.Contains(err.Error(), "unmarshal coverage config:") {
		t.Fatalf("error = %v, want decode error", err)
	}

	_, err = remapConfigAliases([]byte(`{"test_args":["-race"],"testArgs":["-race"]}`))
	if err == nil || !strings.Contains(err.Error(), "remap test args keys:") {
		t.Fatalf("error = %v, want ambiguous remap error", err)
	}

	_, err = remapConfigAliasesWith([]byte(`{"test_args":["-race"]}`), func(any) ([]byte, error) {
		return nil, errBoom
	})
	if err == nil || !strings.Contains(err.Error(), "remap test args keys:") {
		t.Fatalf("error = %v, want marshal remap error", err)
	}
}

func TestResolveNilInputUsesDefaults(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve(nil, nil)
	if err != nil {
		t.Fatalf("Resolve(nil): %v", err)
	}

	if len(resolved.Patterns) == 0 {
		t.Fatal("expected default patterns")
	}
}

func TestUnmarshalRejectsInvalidJSONAndBadRuleTypes(t *testing.T) {
	t.Parallel()

	var cfg Config

	err := Unmarshal([]byte(`{`), &cfg)
	if err == nil || !strings.Contains(err.Error(), "unmarshal coverage config:") {
		t.Fatalf("error = %v, want unmarshal error", err)
	}

	err = Unmarshal([]byte(`{"rules":"nope"}`), &cfg)
	if err == nil || !strings.Contains(err.Error(), "unmarshal coverage config:") {
		t.Fatalf("error = %v, want type mismatch error", err)
	}
}

func TestResolveRejectsAbsolutePathFailures(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	version = func(string) (string, error) {
		return emptyString, errBoom
	}

	t.Run("test result path", func(t *testing.T) {
		_, err := Resolve(&Config{TestResultPath: "reports/test.txt"}, nil)
		if err == nil {
			t.Fatal("expected error")
		}

		msg := err.Error()
		if !strings.Contains(msg, "resolve coverage settings:") {
			t.Fatalf("error = %v, want resolve coverage settings wrap", err)
		}

		if !strings.Contains(msg, "absolute path") {
			t.Fatalf("error = %v, want absolute path failure", err)
		}
	})

	t.Run("coverage result path", func(t *testing.T) {
		_, err := Resolve(&Config{CoverageResultPath: "coverage.out"}, nil)
		if err == nil {
			t.Fatal("expected error")
		}

		msg := err.Error()
		if !strings.Contains(msg, "resolve coverage settings:") {
			t.Fatalf("error = %v, want resolve coverage settings wrap", err)
		}

		if !strings.Contains(msg, "absolute path") {
			t.Fatalf("error = %v, want absolute path failure", err)
		}
	})
}
