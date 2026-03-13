package parser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractSymbolWithTypeDisambiguatesMatches(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "symbols.py", "def Foo():\n    return 1\n\nclass Foo:\n    pass\n")

	got, err := ExtractSymbolWithType(filePath, "Foo", "python", "class")
	if err != nil {
		t.Fatalf("ExtractSymbolWithType() error = %v", err)
	}

	if got.Code != "class Foo:\n    pass" {
		t.Fatalf("ExtractSymbolWithType() gotCode = %q, want %q", got.Code, "class Foo:\n    pass")
	}
	if got.SymbolType != "class" {
		t.Fatalf("ExtractSymbolWithType() gotType = %q, want %q", got.SymbolType, "class")
	}
}

func TestExtractSymbolInfersLanguageFromExtension(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "infer.js", "function hello() { return 1; }\n")

	got, err := ExtractSymbol(filePath, "hello", "")
	if err != nil {
		t.Fatalf("ExtractSymbol() error = %v", err)
	}

	if got.Code != "function hello() { return 1; }" {
		t.Fatalf("ExtractSymbol() gotCode = %q, want %q", got.Code, "function hello() { return 1; }")
	}
}

func TestExtractSymbolErrors(t *testing.T) {
	t.Parallel()

	t.Run("unsupported language", func(t *testing.T) {
		t.Parallel()

		filePath := writeTempFile(t, "unsupported.txt", "hello")
		_, err := ExtractSymbol(filePath, "hello", "")
		if err == nil {
			t.Fatal("expected unsupported language error")
		}
		if !strings.Contains(err.Error(), "unsupported language") {
			t.Fatalf("expected unsupported language error, got: %v", err)
		}
	})

	t.Run("symbol not found", func(t *testing.T) {
		t.Parallel()

		filePath := writeTempFile(t, "missing.go", "package main\n\nfunc Existing() {}\n")
		_, err := ExtractSymbol(filePath, "Missing", "go")
		if err == nil {
			t.Fatal("expected symbol not found error")
		}
		if !errors.Is(err, ErrSymbolNotFound) {
			t.Fatalf("expected ErrSymbolNotFound, got: %v", err)
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected symbol not found error, got: %v", err)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		t.Parallel()

		filePath := writeTempFile(t, "types.py", "def thing():\n    return 1\n")
		_, err := ExtractSymbolWithType(filePath, "thing", "python", "module")
		if err == nil {
			t.Fatal("expected unsupported symbol type error")
		}
		if !strings.Contains(err.Error(), "unsupported symbol type") {
			t.Fatalf("expected unsupported symbol type error, got: %v", err)
		}
	})
}

func TestExtractSymbolSupportsAdvertisedLanguages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fileName   string
		content    string
		symbolName string
		lang       string
		wantCode   string
		wantType   string
	}{
		{
			name:       "Java class",
			fileName:   "sample.java",
			content:    "class Greeter { void wave() {} }\n",
			symbolName: "Greeter",
			lang:       "java",
			wantCode:   "class Greeter { void wave() {} }",
			wantType:   "class",
		},
		{
			name:       "JavaScript function",
			fileName:   "sample.js",
			content:    "function hello() { return 1; }\n",
			symbolName: "hello",
			lang:       "javascript",
			wantCode:   "function hello() { return 1; }",
			wantType:   "function",
		},
		{
			name:       "TypeScript interface",
			fileName:   "sample.ts",
			content:    "interface Person { name: string }\n",
			symbolName: "Person",
			lang:       "typescript",
			wantCode:   "interface Person { name: string }",
			wantType:   "interface",
		},
		{
			name:       "Rust struct",
			fileName:   "sample.rs",
			content:    "struct User { id: i32 }\n",
			symbolName: "User",
			lang:       "rust",
			wantCode:   "struct User { id: i32 }",
			wantType:   "struct",
		},
		{
			name:       "C++ function",
			fileName:   "sample.cpp",
			content:    "int add() { return 1; }\n",
			symbolName: "add",
			lang:       "cpp",
			wantCode:   "int add() { return 1; }",
			wantType:   "function",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filePath := writeTempFile(t, tt.fileName, tt.content)
			got, err := ExtractSymbol(filePath, tt.symbolName, tt.lang)
			if err != nil {
				t.Fatalf("ExtractSymbol() error = %v", err)
			}

			if got.Code != tt.wantCode {
				t.Fatalf("ExtractSymbol() gotCode = %q, want %q", got.Code, tt.wantCode)
			}
			if got.SymbolType != tt.wantType {
				t.Fatalf("ExtractSymbol() gotType = %q, want %q", got.SymbolType, tt.wantType)
			}
		})
	}
}

func TestListSymbols(t *testing.T) {
	t.Parallel()

	content := "package main\n\nfunc Hello() {}\n\nfunc World() {}\n\ntype Config struct {\n\tName string\n}\n"
	filePath := writeTempFile(t, "list.go", content)

	symbols, err := ListSymbols(filePath, "go", "")
	if err != nil {
		t.Fatalf("ListSymbols() error = %v", err)
	}

	if len(symbols) != 3 {
		t.Fatalf("ListSymbols() got %d symbols, want 3", len(symbols))
	}

	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	for _, want := range []string{"Hello", "World", "Config"} {
		if !names[want] {
			t.Fatalf("ListSymbols() missing symbol %q", want)
		}
	}
}

func TestListSymbolsWithTypeFilter(t *testing.T) {
	t.Parallel()

	content := "package main\n\nfunc Hello() {}\n\ntype Config struct {\n\tName string\n}\n"
	filePath := writeTempFile(t, "filter.go", content)

	symbols, err := ListSymbols(filePath, "go", "function")
	if err != nil {
		t.Fatalf("ListSymbols() error = %v", err)
	}

	if len(symbols) != 1 {
		t.Fatalf("ListSymbols() got %d symbols, want 1", len(symbols))
	}
	if symbols[0].Name != "Hello" {
		t.Fatalf("ListSymbols() got name %q, want %q", symbols[0].Name, "Hello")
	}
}

func TestListSymbolsEmptyFile(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "empty.go", "package main\n")

	symbols, err := ListSymbols(filePath, "go", "")
	if err != nil {
		t.Fatalf("ListSymbols() error = %v", err)
	}

	if len(symbols) != 0 {
		t.Fatalf("ListSymbols() got %d symbols, want 0", len(symbols))
	}
}

func TestListSymbolsUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "unsupported.txt", "hello")
	_, err := ListSymbols(filePath, "", "")
	if err == nil {
		t.Fatal("expected unsupported language error")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("expected unsupported language error, got: %v", err)
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file %q: %v", filePath, err)
	}

	return filePath
}
