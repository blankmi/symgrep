package parser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type SymbolInfo struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	FilePath   string `json:"file_path"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	StartByte  uint32 `json:"start_byte"`
	EndByte    uint32 `json:"end_byte"`
	SymbolType string `json:"symbol_type,omitempty"`
}

var ErrSymbolNotFound = errors.New("symbol not found")

var queries = map[string]string{
	"go": `
		[
			(function_declaration name: (identifier) @name)
			(method_declaration name: (field_identifier) @name)
			(type_declaration (type_spec name: (type_identifier) @name)) @symbol
		] @symbol
	`,
	"python": `
		[
			(function_definition name: (identifier) @name)
			(class_definition name: (identifier) @name)
		] @symbol
	`,
	"java": `
		[
			(method_declaration name: (identifier) @name)
			(class_declaration name: (identifier) @name)
			(interface_declaration name: (identifier) @name)
		] @symbol
	`,
	"javascript": `
		[
			(function_declaration name: (identifier) @name)
			(method_definition name: (property_identifier) @name)
			(class_declaration name: (identifier) @name)
		] @symbol
	`,
	"typescript": `
		[
			(function_declaration name: (identifier) @name)
			(method_definition name: (property_identifier) @name)
			(class_declaration name: (type_identifier) @name)
			(interface_declaration name: (type_identifier) @name)
		] @symbol
	`,
	"rust": `
		[
			(function_item name: (identifier) @name)
			(struct_item name: (type_identifier) @name)
			(enum_item name: (type_identifier) @name)
			(trait_item name: (type_identifier) @name)
			(impl_item type: (type_identifier) @name)
		] @symbol
	`,
	"cpp": `
		[
			(function_definition declarator: (function_declarator declarator: (identifier) @name))
			(class_specifier name: (type_identifier) @name)
			(struct_specifier name: (type_identifier) @name)
		] @symbol
	`,
	"xml": `
		[
			(element (STag (Name) @name))
			(element (EmptyElemTag (Name) @name))
		] @symbol
	`,
	"html": `
		[
			(element (start_tag (tag_name) @name))
			(script_element (start_tag (tag_name) @name))
			(style_element (start_tag (tag_name) @name))
			(self_closing_tag (tag_name) @name)
		] @symbol
	`,
}

func ExtractSymbol(filePath, symbolName, langName string) (*SymbolInfo, error) {
	return ExtractSymbolWithType(filePath, symbolName, langName, "")
}

type parsedFile struct {
	content []byte
	langKey string
	tree    *sitter.Tree
	query   *sitter.Query
	cursor  *sitter.QueryCursor
}

func (pf *parsedFile) close() {
	if pf.cursor != nil {
		pf.cursor.Close()
	}
	if pf.query != nil {
		pf.query.Close()
	}
	if pf.tree != nil {
		pf.tree.Close()
	}
}

func parseFileAndQuery(filePath, langName string) (*parsedFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lang, queryKey := GetLanguage(langName, filePath)
	if lang == nil {
		return nil, fmt.Errorf("unsupported language for file: %s", filePath)
	}

	p := sitter.NewParser()
	defer p.Close()

	if err := p.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := p.ParseCtx(context.Background(), content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file: parser returned nil tree")
	}

	queryString, ok := queries[queryKey]
	if !ok {
		return nil, fmt.Errorf("no queries defined for language: %s", queryKey)
	}

	query, queryErr := sitter.NewQuery(lang, queryString)
	if queryErr != nil {
		tree.Close()
		return nil, fmt.Errorf("failed to create query: %w", queryErr)
	}

	cursor := sitter.NewQueryCursor()

	return &parsedFile{
		content: content,
		langKey: queryKey,
		tree:    tree,
		query:   query,
		cursor:  cursor,
	}, nil
}

func ExtractSymbolWithType(filePath, symbolName, langName, symbolType string) (*SymbolInfo, error) {
	normalizedSymbolType, err := normalizeSymbolType(symbolType)
	if err != nil {
		return nil, err
	}

	pf, err := parseFileAndQuery(filePath, langName)
	if err != nil {
		return nil, err
	}
	defer pf.close()

	captureNames := pf.query.CaptureNames()
	matches := pf.cursor.Matches(pf.query, pf.tree.RootNode(), pf.content)

	for match := matches.Next(); match != nil; match = matches.Next() {
		var targetNode sitter.Node
		var nameNode sitter.Node
		hasTarget := false
		hasName := false
		isMatch := false

		for _, capture := range match.Captures {
			if int(capture.Index) >= len(captureNames) {
				continue
			}
			captureName := captureNames[capture.Index]
			capturedNode := capture.Node

			if captureName == "name" {
				if string(pf.content[capturedNode.StartByte():capturedNode.EndByte()]) == symbolName {
					isMatch = true
					nameNode = capturedNode
					hasName = true
				}
			}
			if captureName == "symbol" {
				targetNode = capturedNode
				hasTarget = true
			}
		}

		if isMatch && hasTarget {
			var nameNodePtr *sitter.Node
			if hasName {
				nameNodePtr = &nameNode
			}

			matchedType := inferSymbolType(pf.langKey, &targetNode, nameNodePtr)
			if normalizedSymbolType != "" && !symbolTypeMatches(normalizedSymbolType, matchedType) {
				continue
			}

			name := symbolName
			return &SymbolInfo{
				Name:       name,
				Code:       string(pf.content[targetNode.StartByte():targetNode.EndByte()]),
				FilePath:   filePath,
				StartLine:  int(targetNode.StartPosition().Row) + 1,
				EndLine:    int(targetNode.EndPosition().Row) + 1,
				StartByte:  uint32(targetNode.StartByte()),
				EndByte:    uint32(targetNode.EndByte()),
				SymbolType: matchedType,
			}, nil
		}
	}

	if normalizedSymbolType != "" {
		return nil, fmt.Errorf("%w: symbol '%s' with type '%s'", ErrSymbolNotFound, symbolName, normalizedSymbolType)
	}

	return nil, fmt.Errorf("%w: symbol '%s'", ErrSymbolNotFound, symbolName)
}

func ListSymbols(filePath, langName, symbolType string) ([]SymbolInfo, error) {
	normalizedSymbolType, err := normalizeSymbolType(symbolType)
	if err != nil {
		return nil, err
	}

	pf, err := parseFileAndQuery(filePath, langName)
	if err != nil {
		return nil, err
	}
	defer pf.close()

	captureNames := pf.query.CaptureNames()
	matches := pf.cursor.Matches(pf.query, pf.tree.RootNode(), pf.content)

	seen := make(map[uint]bool)
	var results []SymbolInfo

	for match := matches.Next(); match != nil; match = matches.Next() {
		var targetNode sitter.Node
		var nameNode sitter.Node
		hasTarget := false
		hasName := false

		for _, capture := range match.Captures {
			if int(capture.Index) >= len(captureNames) {
				continue
			}
			captureName := captureNames[capture.Index]
			capturedNode := capture.Node

			if captureName == "name" {
				nameNode = capturedNode
				hasName = true
			}
			if captureName == "symbol" {
				targetNode = capturedNode
				hasTarget = true
			}
		}

		if !hasTarget || !hasName {
			continue
		}

		if seen[targetNode.StartByte()] {
			continue
		}

		matchedType := inferSymbolType(pf.langKey, &targetNode, &nameNode)
		if normalizedSymbolType != "" && !symbolTypeMatches(normalizedSymbolType, matchedType) {
			continue
		}

		seen[targetNode.StartByte()] = true
		name := string(pf.content[nameNode.StartByte():nameNode.EndByte()])

		results = append(results, SymbolInfo{
			Name:       name,
			Code:       string(pf.content[targetNode.StartByte():targetNode.EndByte()]),
			FilePath:   filePath,
			StartLine:  int(targetNode.StartPosition().Row) + 1,
			EndLine:    int(targetNode.EndPosition().Row) + 1,
			StartByte:  uint32(targetNode.StartByte()),
			EndByte:    uint32(targetNode.EndByte()),
			SymbolType: matchedType,
		})
	}

	return results, nil
}

var symbolTypeAliases = map[string]string{
	"":          "",
	"func":      "function",
	"function":  "function",
	"method":    "method",
	"class":     "class",
	"interface": "interface",
	"struct":    "struct",
	"type":      "type",
	"enum":      "enum",
	"trait":     "trait",
	"impl":      "impl",
	"element":   "element",
	"script":    "script",
	"style":     "style",
}

func normalizeSymbolType(symbolType string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(symbolType))
	value, ok := symbolTypeAliases[normalized]
	if !ok {
		return "", fmt.Errorf("unsupported symbol type %q", symbolType)
	}

	return value, nil
}

// symbolTypeMatches reports whether matchedType satisfies the requested filter.
// "function" matches both "function" and "method"; all other types require exact match.
func symbolTypeMatches(filter, matchedType string) bool {
	if filter == matchedType {
		return true
	}
	if filter == "function" && matchedType == "method" {
		return true
	}
	return false
}

func inferSymbolType(langKey string, targetNode, nameNode *sitter.Node) string {
	if targetNode == nil {
		return ""
	}

	if langKey == "go" {
		return inferGoSymbolType(targetNode, nameNode)
	}

	return mapNodeTypeToSymbolType(targetNode.Kind())
}

func inferGoSymbolType(targetNode, nameNode *sitter.Node) string {
	switch targetNode.Kind() {
	case "function_declaration":
		return "function"
	case "method_declaration":
		return "method"
	case "type_declaration":
		if nameNode != nil {
			if typeSpec := nameNode.Parent(); typeSpec != nil && typeSpec.Kind() == "type_spec" {
				if typeNode := typeSpec.ChildByFieldName("type"); typeNode != nil {
					switch typeNode.Kind() {
					case "struct_type":
						return "struct"
					case "interface_type":
						return "interface"
					}
				}
			}
		}
		return "type"
	default:
		return mapNodeTypeToSymbolType(targetNode.Kind())
	}
}

func mapNodeTypeToSymbolType(nodeType string) string {
	switch nodeType {
	case "function_declaration", "function_definition", "function_item", "function_definition_item":
		return "function"
	case "method_declaration", "method_definition":
		return "method"
	case "class_declaration", "class_definition", "class_specifier":
		return "class"
	case "interface_declaration":
		return "interface"
	case "struct_item", "struct_specifier":
		return "struct"
	case "type_declaration", "type_spec":
		return "type"
	case "enum_item":
		return "enum"
	case "trait_item":
		return "trait"
	case "impl_item":
		return "impl"
	case "element", "self_closing_tag":
		return "element"
	case "script_element":
		return "script"
	case "style_element":
		return "style"
	default:
		return ""
	}
}
