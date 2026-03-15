package parser

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeWalkFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func TestWalkFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goFile := filepath.Join(root, "main.go")
	pyFile := filepath.Join(root, "script.py")
	txtFile := filepath.Join(root, "notes.txt")

	writeWalkFile(t, goFile, "package main\n")
	writeWalkFile(t, pyFile, "print('ok')\n")
	writeWalkFile(t, txtFile, "ignore me\n")

	got, err := WalkFiles(root, "")
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}

	want := []string{goFile, pyFile}
	if !slices.Equal(got, want) {
		t.Fatalf("WalkFiles() got %v, want %v", got, want)
	}
}

func TestWalkFilesSkipsDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeWalkFile(t, filepath.Join(root, ".git", "hidden.go"), "package main\n")
	writeWalkFile(t, filepath.Join(root, "node_modules", "dep.go"), "package main\n")
	okFile := filepath.Join(root, "src", "ok.go")
	writeWalkFile(t, okFile, "package main\n")

	got, err := WalkFiles(root, "go")
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}

	want := []string{okFile}
	if !slices.Equal(got, want) {
		t.Fatalf("WalkFiles() got %v, want %v", got, want)
	}
}

func TestWalkFilesLangFilter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goFile := filepath.Join(root, "main.go")
	writeWalkFile(t, goFile, "package main\n")
	writeWalkFile(t, filepath.Join(root, "script.py"), "print('ok')\n")

	got, err := WalkFiles(root, "golang")
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}

	want := []string{goFile}
	if !slices.Equal(got, want) {
		t.Fatalf("WalkFiles() got %v, want %v", got, want)
	}
}

func TestWalkFilesUnsupportedLangFilter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := WalkFiles(root, "ruby")
	if err == nil {
		t.Fatal("expected unsupported language filter error")
	}
}

func TestWalkFilesEmptyDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	got, err := WalkFiles(root, "")
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("WalkFiles() got %v, want empty list", got)
	}
}

func TestWalkFilesNoSupportedFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeWalkFile(t, filepath.Join(root, "readme.txt"), "hello\n")

	got, err := WalkFiles(root, "")
	if err != nil {
		t.Fatalf("WalkFiles() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("WalkFiles() got %v, want empty list", got)
	}
}

func TestIsSupportedFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: "main.go", want: true},
		{path: "script.py", want: true},
		{path: "types.tsx", want: true},
		{path: "page.html", want: true},
		{path: "page.htm", want: true},
		{path: "data.xml", want: true},
		{path: "icon.svg", want: true},
		{path: "readme.md", want: false},
		{path: "noext", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := IsSupportedFile(tt.path); got != tt.want {
				t.Fatalf("IsSupportedFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
