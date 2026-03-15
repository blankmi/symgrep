package parser

import (
	"path/filepath"
	"strings"

	tree_sitter_xml "github.com/tree-sitter-grammars/tree-sitter-xml/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_html "github.com/tree-sitter/tree-sitter-html/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

var extToLang = map[string]string{
	".go":   "go",
	".java": "java",
	".js":   "javascript",
	".jsx":  "javascript",
	".ts":   "typescript",
	".tsx":  "typescript",
	".py":   "python",
	".rs":   "rust",
	".cpp":  "cpp",
	".cc":   "cpp",
	".cxx":  "cpp",
	".h":    "cpp",
	".hpp":  "cpp",
	".xml":  "xml",
	".svg":  "xml",
	".html": "html",
	".htm":  "html",
}

var langAliases = map[string]string{
	"go":         "go",
	"golang":     "go",
	"java":       "java",
	"javascript": "javascript",
	"js":         "javascript",
	"typescript": "typescript",
	"ts":         "typescript",
	"python":     "python",
	"py":         "python",
	"rust":       "rust",
	"rs":         "rust",
	"cpp":        "cpp",
	"c++":        "cpp",
	"xml":        "xml",
	"svg":        "xml",
	"html":       "html",
	"htm":        "html",
}

func normalizeLanguageName(langName string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(langName))
	if normalized == "" {
		return "", false
	}

	canonical, ok := langAliases[normalized]
	return canonical, ok
}

func inferLanguageFromPath(filePath string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(filePath))
	lang, ok := extToLang[ext]
	return lang, ok
}

func GetLanguage(langName, filePath string) (*sitter.Language, string) {
	if strings.TrimSpace(langName) == "" {
		if inferred, ok := inferLanguageFromPath(filePath); ok {
			langName = inferred
		}
	}

	langName, ok := normalizeLanguageName(langName)
	if !ok {
		return nil, ""
	}

	switch langName {
	case "go":
		return sitter.NewLanguage(tree_sitter_go.Language()), "go"
	case "java":
		return sitter.NewLanguage(tree_sitter_java.Language()), "java"
	case "javascript":
		return sitter.NewLanguage(tree_sitter_javascript.Language()), "javascript"
	case "typescript":
		return sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript()), "typescript"
	case "python":
		return sitter.NewLanguage(tree_sitter_python.Language()), "python"
	case "rust":
		return sitter.NewLanguage(tree_sitter_rust.Language()), "rust"
	case "cpp":
		return sitter.NewLanguage(tree_sitter_cpp.Language()), "cpp"
	case "xml":
		return sitter.NewLanguage(tree_sitter_xml.LanguageXML()), "xml"
	case "html":
		return sitter.NewLanguage(tree_sitter_html.Language()), "html"
	default:
		return nil, ""
	}
}
