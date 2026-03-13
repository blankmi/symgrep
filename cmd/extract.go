package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"symgrep/parser"
)

var (
	filePath   string
	symbolName string
	lang       string
	format     string
	symbolType string
)

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract a code symbol from a file or directory",
	Long:  `Extract a function, class, or method by name from a source file or recursively across a directory using Tree-sitter.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExtract(cmd.OutOrStdout(), cmd.ErrOrStderr(), filePath, symbolName, lang, format, symbolType)
	},
}

func runExtract(out, errOut io.Writer, filePath, symbolName, lang, format, symbolType string) error {
	normalizedFormat, err := normalizeFormat(format)
	if err != nil {
		return err
	}

	files, isDir, err := resolveFiles(filePath, lang)
	if err != nil {
		return err
	}

	if !isDir {
		info, err := parser.ExtractSymbolWithType(files[0], symbolName, lang, symbolType)
		if err != nil {
			return err
		}

		switch normalizedFormat {
		case "json":
			data, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			_, err = fmt.Fprintln(out, string(data))
			return err
		case "raw":
			_, err = fmt.Fprint(out, info.Code)
			return err
		default:
			return fmt.Errorf("invalid format %q", format)
		}
	}

	var (
		matches        []parser.SymbolInfo
		failedFiles    int
		processedFiles int
	)

	for _, fp := range files {
		info, extractErr := parser.ExtractSymbolWithType(fp, symbolName, lang, symbolType)
		if extractErr != nil {
			if errors.Is(extractErr, parser.ErrSymbolNotFound) {
				processedFiles++
				continue
			}

			failedFiles++
			if errOut != nil {
				_, _ = fmt.Fprintf(errOut, "warning: failed to extract symbol in %s: %v\n", fp, extractErr)
			}
			continue
		}

		processedFiles++
		matches = append(matches, *info)
	}

	if processedFiles == 0 && failedFiles > 0 {
		return fmt.Errorf("failed to process any files under %s (%d errors)", filePath, failedFiles)
	}

	if len(matches) == 0 {
		return fmt.Errorf("symbol %q not found in any file under %s", symbolName, filePath)
	}

	switch normalizedFormat {
	case "json":
		data, err := json.MarshalIndent(matches, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = fmt.Fprintln(out, string(data))
		return err
	case "raw":
		for i, info := range matches {
			if i > 0 {
				if _, err := fmt.Fprint(out, "\n\n"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(out, "=== file: %s ===\n%s", info.FilePath, info.Code); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("invalid format %q", format)
	}
}

func normalizeFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "raw":
		return "raw", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("invalid format %q: expected one of raw, json", format)
	}
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().StringVarP(&filePath, "file", "f", "", "The target file or directory path")
	extractCmd.Flags().StringVarP(&symbolName, "symbol", "s", "", "The name of the function, class, or method to extract")
	extractCmd.Flags().StringVarP(&lang, "lang", "l", "", "The language of the target files (optional, inferred from extension if omitted)")
	extractCmd.Flags().StringVar(&format, "format", "raw", "Output format (raw, json)")
	extractCmd.Flags().StringVarP(&symbolType, "type", "t", "", "Filter by symbol type (e.g., function, class, struct)")

	_ = extractCmd.MarkFlagRequired("file")
	_ = extractCmd.MarkFlagRequired("symbol")
}
