package cmd

import (
	"fmt"
	"os"

	"symgrep/parser"
)

func resolveFiles(path, langFilter string) (files []string, isDir bool, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false, err
	}

	if !info.IsDir() {
		return []string{path}, false, nil
	}

	files, err = parser.WalkFiles(path, langFilter)
	if err != nil {
		return nil, true, err
	}

	if len(files) == 0 {
		if langFilter != "" {
			return nil, true, fmt.Errorf("no supported files found under %s for language %q", path, langFilter)
		}
		return nil, true, fmt.Errorf("no supported files found under %s", path)
	}

	return files, true, nil
}
