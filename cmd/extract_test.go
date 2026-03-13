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

func TestRunExtractRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runExtract(&out, &errOut, "unused", "unused", "go", "yaml", "")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("expected invalid format error, got: %v", err)
	}
}

func TestRunExtractAppliesSymbolTypeFilter(t *testing.T) {
	t.Parallel()

	content := "def Foo():\n    return 1\n\nclass Foo:\n    pass\n"
	filePath := filepath.Join(t.TempDir(), "symbols.py")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runExtract(&out, &errOut, filePath, "Foo", "python", "raw", "class")
	if err != nil {
		t.Fatalf("runExtract returned error: %v", err)
	}

	if out.String() != "class Foo:\n    pass" {
		t.Fatalf("unexpected extraction output: %q", out.String())
	}
}

func TestRunExtractDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n\nfunc Shared() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package main\n\nfunc Shared() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := runExtract(&out, &errOut, root, "Shared", "go", "json", ""); err != nil {
		t.Fatalf("runExtract returned error: %v", err)
	}

	var symbols []parser.SymbolInfo
	if err := json.Unmarshal(out.Bytes(), &symbols); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(symbols))
	}
}

func TestRunExtractDirectoryNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n\nfunc One() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	err := runExtract(&out, &errOut, root, "Missing", "go", "json", "")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "not found in any file under") {
		t.Fatalf("expected not found under dir error, got: %v", err)
	}
}

func TestRunExtractSingleFileBackwardCompat(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "single.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc One() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := runExtract(&out, &errOut, filePath, "One", "go", "json", ""); err != nil {
		t.Fatalf("runExtract returned error: %v", err)
	}

	var symbol parser.SymbolInfo
	if err := json.Unmarshal(out.Bytes(), &symbol); err != nil {
		t.Fatalf("expected single-object JSON for single-file mode, got error: %v", err)
	}
	if symbol.Name != "One" {
		t.Fatalf("expected symbol One, got %q", symbol.Name)
	}
}

func TestRunExtractDirectoryRawUsesNeutralBanner(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n\nfunc Shared() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := runExtract(&out, &errOut, root, "Shared", "go", "raw", ""); err != nil {
		t.Fatalf("runExtract returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "=== file: ") {
		t.Fatalf("expected neutral file banner, got: %q", output)
	}
	if !strings.Contains(output, "func Shared() {}") {
		t.Fatalf("expected extracted code, got: %q", output)
	}
}

func TestRunExtractDirectoryGuardedFailureWhenAllFilesFail(t *testing.T) {
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
	err := runExtract(&out, &errOut, root, "Bad", "go", "raw", "")
	if err == nil {
		t.Fatal("expected guarded failure error")
	}
	if !strings.Contains(err.Error(), "failed to process any files under") {
		t.Fatalf("expected guarded failure error, got: %v", err)
	}
	if !strings.Contains(errOut.String(), "warning: failed to extract symbol in") {
		t.Fatalf("expected warning output, got: %q", errOut.String())
	}
}
