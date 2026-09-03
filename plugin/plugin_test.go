// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package plugin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golangci/plugin-module-register/register"
	"github.com/gostafa/coverlint/analyzer"
	_ "github.com/gostafa/coverlint/plugin"
	"golang.org/x/tools/go/analysis"
)

const (
	settingMin      = "min"
	settingPackages = "packages"
	settingTestArgs = "test-args"
	settingTimeout  = "timeout"
	runFlag         = "-run"
	testAddName     = "TestAdd"
)

func TestRegisteredPluginBuildsAnalyzer(t *testing.T) {
	t.Parallel()

	dir := writePluginFixture(t)

	plug, err := registeredPlugin(t)(fixtureSettings(dir))
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	assertLoadMode(t, plug)

	analyzers := buildAnalyzersForTest(t, plug)
	assertAnalyzer(t, analyzers)
	assertBuildAnalyzersAgain(t, plug)
}

func fixtureSettings(dir string) map[string]any {
	return map[string]any{
		settingMin:      100,
		settingPackages: []string{dir},
		settingTimeout:  time.Minute.String(),
		settingTestArgs: []string{runFlag, testAddName},
	}
}

func assertLoadMode(t *testing.T, plug register.LinterPlugin) {
	t.Helper()

	if got := plug.GetLoadMode(); got != register.LoadModeSyntax {
		t.Fatalf("GetLoadMode() = %q, want %q", got, register.LoadModeSyntax)
	}
}

func buildAnalyzersForTest(t *testing.T, plug register.LinterPlugin) []*analysis.Analyzer {
	t.Helper()

	analyzers, err := plug.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}

	return analyzers
}

func assertAnalyzer(t *testing.T, analyzers []*analysis.Analyzer) {
	t.Helper()

	if len(analyzers) != 1 {
		t.Fatalf("analyzers = %#v, want one analyzer", analyzers)
	}

	if analyzers[0].Name != analyzer.Name {
		t.Fatalf("analyzer name = %q, want %q", analyzers[0].Name, analyzer.Name)
	}

	if analyzers[0].ResultType == nil {
		t.Fatal("ResultType is nil")
	}
}

func assertBuildAnalyzersAgain(t *testing.T, plug register.LinterPlugin) {
	t.Helper()

	again, err := plug.BuildAnalyzers()
	if err != nil {
		t.Fatalf("second BuildAnalyzers: %v", err)
	}

	if again[0].Run == nil {
		t.Fatal("analyzer Run is nil")
	}
}

func TestRegisteredPluginBuildAnalyzersReturnsLoadError(t *testing.T) {
	t.Parallel()

	constructor := registeredPlugin(t)

	plug, err := constructor(map[string]any{
		settingMin: 101,
	})
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	_, err = plug.BuildAnalyzers()

	if err == nil || !strings.Contains(err.Error(), "run coverlint") {
		t.Fatalf("error = %v, want load wrapper", err)
	}
}

func TestRegisteredPluginWrapsDecodeError(t *testing.T) {
	t.Parallel()

	constructor := registeredPlugin(t)

	_, err := constructor("not settings")

	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("error = %v, want decode wrapper", err)
	}
}

func TestRegisteredPluginRejectsUnknownSettings(t *testing.T) {
	t.Parallel()

	constructor := registeredPlugin(t)

	_, err := constructor(map[string]any{
		"minimum": 85,
	})

	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error = %v, want unknown setting error", err)
	}
}

func registeredPlugin(t *testing.T) register.NewPlugin {
	t.Helper()

	constructor, err := register.GetPlugin(analyzer.Name)
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}

	return constructor
}

func writePluginFixture(t *testing.T) string {
	t.Helper()

	dir := moduleFixtureDir(t)

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("RemoveAll %s: %v", dir, err)
		}
	})

	writePluginFile(
		t,
		dir,
		"calc.go",
		"package pluginfixture\n\nfunc Add(a, b int) int { return a + b }\nfunc Missing() int { return 1 }\n",
	)
	writePluginFile(t, dir, "calc_test.go", `package pluginfixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`)

	return "./" + filepath.Base(dir)
}

func writePluginFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
	if err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}

func fixtureName(t *testing.T) string {
	t.Helper()

	return strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
}

func moduleFixtureDir(t *testing.T) string {
	t.Helper()

	temp := t.TempDir()
	suffix := filepath.Base(filepath.Dir(temp)) + "-" + filepath.Base(temp)
	dir := filepath.Join(".", "coverlint-fixture-"+fixtureName(t)+"-"+suffix)

	err := os.Mkdir(dir, 0o700)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	return dir
}
