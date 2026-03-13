package parser

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

var skipDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".idea":        true,
	".vscode":      true,
}

func IsSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := extToLang[ext]
	return ok
}

func WalkFiles(root, langFilter string) ([]string, error) {
	var canonicalFilter string
	if strings.TrimSpace(langFilter) != "" {
		filter, ok := normalizeLanguageName(langFilter)
		if !ok {
			return nil, fmt.Errorf("unsupported language filter %q", langFilter)
		}
		canonicalFilter = filter
	}

	files := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path == root {
				return nil
			}
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.Type().IsRegular() || !IsSupportedFile(path) {
			return nil
		}

		if canonicalFilter != "" {
			fileLang, ok := inferLanguageFromPath(path)
			if !ok || fileLang != canonicalFilter {
				return nil
			}
		}

		files = append(files, path)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to walk files under %s: %w", root, err)
	}

	sort.Strings(files)
	return files, nil
}
