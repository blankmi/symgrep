package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"symgrep/parser"
)

var (
	listFilePath   string
	listLang       string
	listFormat     string
	listSymbolType string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all symbols in a file or directory",
	Long:  `List all functions, classes, methods, and other symbols defined in a source file or recursively across a directory using Tree-sitter.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd.OutOrStdout(), cmd.ErrOrStderr(), listFilePath, listLang, listFormat, listSymbolType)
	},
}

func runList(out, errOut io.Writer, filePath, lang, format, symbolType string) error {
	normalizedFormat, err := normalizeFormat(format)
	if err != nil {
		return err
	}

	files, isDir, err := resolveFiles(filePath, lang)
	if err != nil {
		return err
	}

	var (
		symbols        []parser.SymbolInfo
		failedFiles    int
		processedFiles int
	)

	for _, fp := range files {
		fileSymbols, listErr := parser.ListSymbols(fp, lang, symbolType)
		if listErr != nil {
			if !isDir {
				return listErr
			}
			failedFiles++
			if errOut != nil {
				_, _ = fmt.Fprintf(errOut, "warning: failed to list symbols in %s: %v\n", fp, listErr)
			}
			continue
		}

		processedFiles++
		symbols = append(symbols, fileSymbols...)
	}

	if isDir && processedFiles == 0 && failedFiles > 0 {
		return fmt.Errorf("failed to process any files under %s (%d errors)", filePath, failedFiles)
	}

	switch normalizedFormat {
	case "json":
		data, err := json.MarshalIndent(symbols, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = fmt.Fprintln(out, string(data))
		return err
	case "raw":
		for _, s := range symbols {
			var err error
			if isDir {
				_, err = fmt.Fprintf(out, "%s\t%s\t%s\tL%d-L%d\n", s.FilePath, s.SymbolType, s.Name, s.StartLine, s.EndLine)
			} else {
				_, err = fmt.Fprintf(out, "%s\t%s\tL%d-L%d\n", s.SymbolType, s.Name, s.StartLine, s.EndLine)
			}
			if err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("invalid format %q", format)
	}
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listFilePath, "file", "f", "", "The target file or directory path")
	listCmd.Flags().StringVarP(&listLang, "lang", "l", "", "The language of the target files (optional, inferred from extension if omitted)")
	listCmd.Flags().StringVar(&listFormat, "format", "raw", "Output format (raw, json)")
	listCmd.Flags().StringVarP(&listSymbolType, "type", "t", "", "Filter by symbol type (e.g., function, class, struct)")

	_ = listCmd.MarkFlagRequired("file")
}
