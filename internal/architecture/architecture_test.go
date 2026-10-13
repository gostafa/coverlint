// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package architecture

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/gostafa/coverlint/"

func TestRepositoryBoundaries(t *testing.T) {
	t.Parallel()
	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && skipDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, finding := range sourceFindings(filepath.ToSlash(relative), file) {
			t.Errorf("%s: %s", relative, finding)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func skipDirectory(name string) bool {
	return slices.Contains(
		[]string{".git", ".task", ".cache", "bin", "vendor", "taskotter"},
		name,
	) ||
		strings.Contains(name, "fixture-")
}

func sourceFindings(path string, file *ast.File) []string {
	findings := []string{}
	name := strings.TrimSuffix(file.Name.Name, "_test")
	allowed := []string{
		"funcs.go",
		"types.go",
		"vars.go",
		"consts.go",
		name + "_test.go",
		name + "_integration_test.go",
		name + "_bench_test.go",
	}
	base := filepath.Base(path)
	if !slices.Contains(allowed, base) {
		findings = append(findings, "Go filename is outside the package convention")
	}
	isTest := strings.HasSuffix(base, "_test.go")
	if isTest {
		if strings.HasSuffix(base, "_integration_test.go") &&
			!strings.HasSuffix(file.Name.Name, "_test") {
			findings = append(findings, "integration tests must use an external package")
		}
		if !strings.HasSuffix(base, "_integration_test.go") &&
			strings.HasSuffix(file.Name.Name, "_test") {
			findings = append(findings, "implementation tests must use the implementation package")
		}
		return findings
	}
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if base != "funcs.go" {
				findings = append(findings, "function outside funcs.go")
			}
		case *ast.GenDecl:
			wanted := map[token.Token]string{token.TYPE: "types.go", token.CONST: "consts.go", token.VAR: "vars.go"}[value.Tok]
			if wanted != "" && wanted != base {
				findings = append(findings, "declaration outside "+wanted)
			}
		}
	}
	for _, imported := range file.Imports {
		dependency, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			findings = append(findings, err.Error())
			continue
		}
		if reason := importViolation(path, dependency); reason != "" {
			findings = append(findings, reason+": "+dependency)
		}
	}
	return findings
}

func feature(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 2 && parts[0] == "internal" && parts[1] == "features" {
		return parts[2]
	}
	return ""
}

func importViolation(source, dependency string) string {
	local := strings.TrimPrefix(dependency, modulePath)
	owner := feature(source)
	provider := feature(local)
	if owner != "" && provider != "" && owner != provider && !strings.HasSuffix(local, "/public") &&
		!strings.Contains(local, "/public/") {
		return "cross-feature private import"
	}
	if strings.HasPrefix(source, "internal/infrastructure/") && provider != "" {
		return "feature-specific infrastructure"
	}
	if strings.Contains(source, "/domain/") {
		if strings.Contains(dependency, ".") &&
			!strings.HasPrefix(local, "internal/features/"+owner+"/domain") {
			return "domain imports non-domain code"
		}
		if slices.Contains(
			[]string{"os", "os/exec", "syscall", "net", "net/http", "database/sql"},
			dependency,
		) {
			return "domain performs external I/O"
		}
	}
	if strings.Contains(source, "/application/") && strings.HasPrefix(dependency, modulePath) {
		allowed := "internal/features/" + owner + "/"
		if !strings.HasPrefix(local, allowed+"domain") &&
			!strings.HasPrefix(local, allowed+"ports") &&
			!strings.HasPrefix(local, allowed+"application") {
			return "application imports a concrete implementation"
		}
	}
	if strings.Contains(source, "/application/") && strings.Contains(dependency, ".") &&
		!strings.HasPrefix(dependency, modulePath) {
		return "application imports a vendor dependency"
	}
	return ""
}

func TestBoundaryCheckerRejectsViolations(t *testing.T) {
	t.Parallel()
	tests := []struct{ source, dependency string }{
		{
			"internal/features/policy/domain/funcs.go",
			modulePath + "internal/features/collection/domain",
		},
		{
			"internal/features/policy/application/funcs.go",
			modulePath + "internal/features/policy/adapters/files",
		},
		{"internal/features/policy/domain/funcs.go", "os"},
		{"internal/features/policy/domain/funcs.go", "golang.org/x/tools/go/analysis"},
		{"internal/infrastructure/cache/funcs.go", modulePath + "internal/features/policy/ports"},
		{
			"internal/features/policy/adapters/files/funcs.go",
			modulePath + "internal/features/reporting/adapters/gotool",
		},
	}
	for _, test := range tests {
		if importViolation(test.source, test.dependency) == "" {
			t.Errorf("accepted forbidden dependency %s -> %s", test.source, test.dependency)
		}
	}
	file, err := parser.ParseFile(
		token.NewFileSet(),
		"errors.go",
		"package sample\nvar errExample error\nfunc wrong() {}",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(sourceFindings("errors.go", file)) != 3 {
		t.Fatal("filename/declaration checker accepted invalid fixture")
	}
	file, err = parser.ParseFile(
		token.NewFileSet(),
		"sample_integration_test.go",
		"package sample",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(sourceFindings("sample_integration_test.go", file)) != 1 {
		t.Fatal("accepted internal integration test package")
	}
	if importViolation(
		"internal/features/policy/adapters/collection/funcs.go",
		modulePath+"internal/features/collection/public",
	) != "" {
		t.Fatal("rejected published feature contract")
	}
}

func TestConsumerTypesAndImportCycles(t *testing.T) {
	t.Parallel()
	exports, paths := packageExports(t)
	loader := importer.ForCompiler(
		token.NewFileSet(),
		"gc",
		func(path string) (io.ReadCloser, error) { return os.Open(exports[path]) },
	)
	for _, path := range paths {
		if !consumerPackage(path) {
			continue
		}
		pkg, err := loader.Import(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range pkg.Scope().Names() {
			object := pkg.Scope().Lookup(name)
			if object.Exported() {
				checkPublicType(
					object.Type(),
					map[types.Type]bool{},
					path+"."+name,
					func(problem string) { t.Error(problem) },
				)
			}
		}
	}
}

func packageExports(t *testing.T) (map[string]string, []string) {
	t.Helper()
	command := exec.CommandContext(
		context.Background(),
		"go",
		"list",
		"-deps",
		"-export",
		"-json",
		"./...",
	)
	command.Dir = repositoryRoot(t)
	data, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	exports := map[string]string{}
	paths := []string{}
	for decoder.More() {
		var item struct {
			ImportPath string
			Export     string
		}
		if err := decoder.Decode(&item); err != nil {
			t.Fatal(err)
		}
		exports[item.ImportPath] = item.Export
		paths = append(paths, item.ImportPath)
	}
	return exports, paths
}

func consumerPackage(path string) bool {
	return slices.Contains(
		[]string{modulePath + "coverlint", modulePath + "analyzer", modulePath + "plugin"},
		path,
	) ||
		(strings.HasSuffix(path, "/public") || strings.Contains(path, "/public/"))
}

func checkPublicType(
	value types.Type,
	seen map[types.Type]bool,
	label string,
	report func(string),
) {
	if seen[value] {
		return
	}
	seen[value] = true
	switch current := value.(type) {
	case *types.Alias:
		checkPublicType(types.Unalias(current), seen, label, report)
	case *types.Named:
		if current.Obj().Pkg() != nil {
			path := current.Obj().Pkg().Path()
			if path == "golang.org/x/tools/go/analysis" && current.Obj().Name() == "Analyzer" &&
				(strings.HasPrefix(label, modulePath+"analyzer.") || strings.HasPrefix(label, modulePath+"plugin.")) {
				return
			}
			if strings.Contains(path, ".") && !consumerPackage(path) {
				report(label + " exposes " + path)
				return
			}
		}
		checkPublicType(current.Underlying(), seen, label, report)
		for index := range current.NumMethods() {
			if current.Method(index).Exported() {
				checkPublicType(current.Method(index).Type(), seen, label, report)
			}
		}
	case *types.Pointer:
		checkPublicType(current.Elem(), seen, label, report)
	case *types.Slice:
		checkPublicType(current.Elem(), seen, label, report)
	case *types.Array:
		checkPublicType(current.Elem(), seen, label, report)
	case *types.Map:
		checkPublicType(current.Key(), seen, label, report)
		checkPublicType(current.Elem(), seen, label, report)
	case *types.Chan:
		checkPublicType(current.Elem(), seen, label, report)
	case *types.Struct:
		for index := range current.NumFields() {
			if current.Field(index).Exported() {
				checkPublicType(current.Field(index).Type(), seen, label, report)
			}
		}
	case *types.Signature:
		checkPublicType(current.Params(), seen, label, report)
		checkPublicType(current.Results(), seen, label, report)
	case *types.Tuple:
		for index := range current.Len() {
			checkPublicType(current.At(index).Type(), seen, label, report)
		}
	case *types.Interface:
		for index := range current.NumMethods() {
			checkPublicType(current.Method(index).Type(), seen, label, report)
		}
	}
}

func TestPublicTypeCheckerRejectsInternalNamedType(t *testing.T) {
	t.Parallel()
	private := types.NewPackage(modulePath+"internal/features/policy/domain", "domain")
	value := types.NewNamed(
		types.NewTypeName(token.NoPos, private, "Private", nil),
		types.NewStruct(nil, nil),
		nil,
	)
	problems := []string{}
	checkPublicType(
		types.NewPointer(value),
		map[types.Type]bool{},
		modulePath+"coverlint.Fixture",
		func(problem string) { problems = append(problems, problem) },
	)
	if len(problems) != 1 {
		t.Fatalf("private type accepted: %v", problems)
	}
}
