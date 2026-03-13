package parser

import (
	"os"
	"testing"
)

func TestExtractSymbol(t *testing.T) {
	// Sample Go code
	goCode := `
package main
func MyFunc() {
	println("hello")
}
type MyType struct {
	A int
}
`
	goFile := "test_go.go"
	os.WriteFile(goFile, []byte(goCode), 0644)
	defer os.Remove(goFile)

	tests := []struct {
		name       string
		filePath   string
		symbolName string
		lang       string
		wantCode   string
	}{
		{
			name:       "Go function",
			filePath:   goFile,
			symbolName: "MyFunc",
			lang:       "go",
			wantCode:   "func MyFunc() {\n\tprintln(\"hello\")\n}",
		},
		{
			name:       "Go struct",
			filePath:   goFile,
			symbolName: "MyType",
			lang:       "go",
			wantCode:   "type MyType struct {\n\tA int\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractSymbol(tt.filePath, tt.symbolName, tt.lang)
			if err != nil {
				t.Fatalf("ExtractSymbol() error = %v", err)
			}
			if got.Code != tt.wantCode {
				t.Errorf("ExtractSymbol() gotCode = %v, want %v", got.Code, tt.wantCode)
			}
		})
	}
}

func TestExtractSymbol_Python(t *testing.T) {
	pyCode := `
def my_func():
    print("hello")

class MyClass:
    pass
`
	pyFile := "test_py.py"
	os.WriteFile(pyFile, []byte(pyCode), 0644)
	defer os.Remove(pyFile)

	tests := []struct {
		name       string
		filePath   string
		symbolName string
		lang       string
		wantCode   string
	}{
		{
			name:       "Python function",
			filePath:   pyFile,
			symbolName: "my_func",
			lang:       "python",
			wantCode:   "def my_func():\n    print(\"hello\")",
		},
		{
			name:       "Python class",
			filePath:   pyFile,
			symbolName: "MyClass",
			lang:       "python",
			wantCode:   "class MyClass:\n    pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractSymbol(tt.filePath, tt.symbolName, tt.lang)
			if err != nil {
				t.Fatalf("ExtractSymbol() error = %v", err)
			}
			if got.Code != tt.wantCode {
				t.Errorf("ExtractSymbol() gotCode = %v, want %v", got.Code, tt.wantCode)
			}
		})
	}
}
