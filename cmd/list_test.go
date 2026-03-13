package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"symgrep/parser"
)

const goTestFile = `package main

func Hello() {}

func World() {}

type Config struct {
	Name string
}
`

func writeListTestFile(t *testing.T, name, content string) string {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return filePath
}

func TestRunListRawFormat(t *testing.T) {
	t.Parallel()

	fp := writeListTestFile(t, "raw.go", goTestFile)
	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runList(&out, &errOut, fp, "go", "raw", "")
	if err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "function\tHello\t") {
		t.Fatalf("expected Hello function in output, got: %q", output)
	}
	if !strings.Contains(output, "function\tWorld\t") {
		t.Fatalf("expected World function in output, got: %q", output)
	}
	if !strings.Contains(output, "struct\tConfig\t") {
		t.Fatalf("expected Config struct in output, got: %q", output)
	}
}

func TestRunListJsonFormat(t *testing.T) {
	t.Parallel()

	fp := writeListTestFile(t, "json.go", goTestFile)
	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runList(&out, &errOut, fp, "go", "json", "")
	if err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	var symbols []parser.SymbolInfo
	if err := json.Unmarshal(out.Bytes(), &symbols); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(symbols) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(symbols))
	}
}

func TestRunListWithTypeFilter(t *testing.T) {
	t.Parallel()

	fp := writeListTestFile(t, "filter.go", goTestFile)
	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runList(&out, &errOut, fp, "go", "raw", "function")
	if err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	output := out.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 function lines, got %d: %q", len(lines), output)
	}
	if strings.Contains(output, "struct") {
		t.Fatalf("expected no struct in filtered output, got: %q", output)
	}
}

func TestRunListRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runList(&out, &errOut, "unused", "go", "yaml", "")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("expected invalid format error, got: %v", err)
	}
}

func TestRunListDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goFile := filepath.Join(root, "a.go")
	pyFile := filepath.Join(root, "b.py")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc GoFn() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}
	if err := os.WriteFile(pyFile, []byte("def py_fn():\n    return 1\n"), 0o644); err != nil {
		t.Fatalf("failed to write python file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := runList(&out, &errOut, root, "", "json", ""); err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	var symbols []parser.SymbolInfo
	if err := json.Unmarshal(out.Bytes(), &symbols); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(symbols))
	}

	names := make(map[string]bool)
	for _, sym := range symbols {
		names[sym.Name] = true
	}
	if !names["GoFn"] || !names["py_fn"] {
		t.Fatalf("expected GoFn and py_fn symbols, got: %#v", names)
	}
}

func TestRunListDirectoryWithLangFilter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n\nfunc GoFn() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.py"), []byte("def py_fn():\n    return 1\n"), 0o644); err != nil {
		t.Fatalf("failed to write python file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := runList(&out, &errOut, root, "golang", "json", ""); err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	var symbols []parser.SymbolInfo
	if err := json.Unmarshal(out.Bytes(), &symbols); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0].Name != "GoFn" {
		t.Fatalf("expected GoFn symbol, got %q", symbols[0].Name)
	}
}

func TestRunListDirectoryRawIncludesPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n\nfunc GoFn() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := runList(&out, &errOut, root, "", "raw", ""); err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "a.go\tfunction\tGoFn\t") {
		t.Fatalf("expected path-prefixed output, got: %q", output)
	}
}

func TestRunListDirectoryWarnsAndContinuesOnFileError(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("permission-based read errors are not portable on Windows")
	}

	root := t.TempDir()
	good := filepath.Join(root, "good.go")
	bad := filepath.Join(root, "bad.go")

	if err := os.WriteFile(good, []byte("package main\n\nfunc Good() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write good file: %v", err)
	}
	if err := os.WriteFile(bad, []byte("package main\n\nfunc Bad() {}\n"), 0o000); err != nil {
		t.Fatalf("failed to write bad file: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o644) })

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := runList(&out, &errOut, root, "go", "raw", ""); err != nil {
		t.Fatalf("runList returned error: %v", err)
	}

	if !strings.Contains(out.String(), "good.go\tfunction\tGood\t") {
		t.Fatalf("expected symbol from good file, got: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "warning: failed to list symbols in") {
		t.Fatalf("expected warning output, got: %q", errOut.String())
	}
}

func TestRunListDirectoryGuardedFailureWhenAllFilesFail(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("permission-based read errors are not portable on Windows")
	}

	root := t.TempDir()
	bad := filepath.Join(root, "bad.go")
	if err := os.WriteFile(bad, []byte("package main\n\nfunc Bad() {}\n"), 0o000); err != nil {
		t.Fatalf("failed to write bad file: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o644) })

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runList(&out, &errOut, root, "go", "raw", "")
	if err == nil {
		t.Fatal("expected guarded failure error")
	}
	if !strings.Contains(err.Error(), "failed to process any files under") {
		t.Fatalf("expected guarded failure error, got: %v", err)
	}
	if !strings.Contains(errOut.String(), "warning: failed to list symbols in") {
		t.Fatalf("expected warning output, got: %q", errOut.String())
	}
}
