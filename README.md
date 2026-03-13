# symgrep

`symgrep` is a high-precision code symbol extraction tool built for LLM agents and developers. Unlike standard `grep` which operates on text, `symgrep` uses **Tree-sitter** to parse source code into an Abstract Syntax Tree (AST), allowing it to extract exact functions, classes, methods, and structs with zero noise.

## Features

- **Precision Extraction**: Extract symbols by name (e.g., `MyFunction`, `UserDataStruct`) without capturing unrelated text.
- **Symbol Listing**: List all symbols in a file — useful for discovering what's defined before extracting.
- **Directory-Aware Search**: Run `list` and `extract` recursively across directories with deterministic output ordering.
- **Multi-Language Support**: Built-in support for:
  - Go
  - Python
  - Java
  - JavaScript / TypeScript
  - Rust
  - C++
- **Agent-Optimized**: Output in `raw` code format for easy reading or `json` for structured data (includes line numbers and byte offsets).
- **Single Binary**: Portable Go executable with embedded C-based grammars (requires CGO for building).

## Installation

### From Source
You will need Go installed and a C compiler (CGO is required for Tree-sitter).

```bash
git clone https://github.com/youruser/symgrep
cd symgrep
make build
# Binary will be available as ./bin/symgrep
```

*Note: For cross-compilation, the included Makefile uses `zig cc` to handle CGO dependencies easily.*

## Usage

### Basic Extraction
Extract a function from a Go file:
```bash
./bin/symgrep extract -f main.go -s MyFunction
```

### JSON Output
Get structured data for an LLM agent:
```bash
./bin/symgrep extract -f models.py -s UserProfile --format=json
```

**JSON Output Example:**
```json
{
  "name": "UserProfile",
  "code": "class UserProfile:\n    def __init__(self, name):\n        self.name = name",
  "file_path": "models.py",
  "start_line": 12,
  "end_line": 15,
  "start_byte": 450,
  "end_byte": 582
}
```

### Listing Symbols
List all symbols defined in a file:
```bash
./bin/symgrep list -f parser/engine.go
```

**Output:**
```
function    ExtractSymbol       L77-L79
struct      SymbolInfo          L12-L21
function    ListSymbols         L131-L189
```

List as JSON:
```bash
./bin/symgrep list -f parser/engine.go --format=json
```

Filter by type:
```bash
./bin/symgrep list -f parser/engine.go -t function
```

List symbols recursively across a directory:
```bash
./bin/symgrep list -f parser/
```

Extract a symbol recursively across a directory:
```bash
./bin/symgrep extract -f parser/ -s GetLanguage --format=json
```

### `extract` Flags
- `-f, --file`: Path to the source file or directory (Required).
- `-s, --symbol`: Name of the function, class, or method to extract (Required).
- `-l, --lang`: Language of the file (Optional; inferred from extension if omitted).
- `--format`: Output format, either `raw` (default) or `json`.
- `-t, --type`: Filter by symbol type (Optional; e.g., `function`, `class`).

### `list` Flags
- `-f, --file`: Path to the source file or directory (Required).
- `-l, --lang`: Language of the file (Optional; inferred from extension if omitted).
- `--format`: Output format, either `raw` (default) or `json`.
- `-t, --type`: Filter by symbol type (Optional; e.g., `function`, `class`).

## Agent Integration

Pre-built skill files for popular coding agents are in `agent-skills/`:

| Agent | File | How to install |
|-------|------|----------------|
| Claude Code | `agent-skills/claude-code/SKILL.md` | Copy into `CLAUDE.md` or reference via [custom instructions](https://docs.anthropic.com/en/docs/claude-code) |
| Gemini | `agent-skills/gemini/gemini.skill` | `gemini skills install agent-skills/gemini/gemini.skill` |
| Codex | `agent-skills/codex/SKILL.md` | Copy `agent-skills/codex` into `$CODEX_HOME/skills/symgrep` |

### Agent Workflow
The intended two-step pattern for agents:

```bash
# 1. Discover symbols in the target (file or directory)
./bin/symgrep list -f src/
# src/server.go	function	handleRequest	L12-L45
# src/server.go	function	startServer	L47-L89
# src/server.go	struct	Config	L3-L10

# 2. Extract only what you need
./bin/symgrep extract -f src/ -s handleRequest --format=json
```

### Exit Codes & Error Handling
- Exit `0` on success, `1` on any error.
- Errors go to **stderr only** — stdout is always clean parseable output.
- `list` returns an empty result (exit `0`) when a file has no symbols.

### Output Formats

**`list` raw** (default):
- File mode: `type\tname\tLstart-Lend`
- Directory mode: `path\ttype\tname\tLstart-Lend`

File mode example:
```
type	name	Lstart-Lend
```

Directory mode example:
```
src/server.go	function	handleRequest	L12-L45
```

**`list` JSON** — always a JSON array of symbols.

**`extract` JSON**:
- File mode: a single JSON object
- Directory mode: a JSON array of matches

**`extract` raw**:
- File mode: raw symbol source only
- Directory mode: each match is prefixed with `=== file: <path> ===`

### Directory Traversal Rules

- Recursively scans supported source files under the target directory.
- Skips common non-source/vendor/tooling directories such as `.git`, `node_modules`, `vendor`, `__pycache__`, `.idea`, and `.vscode`.
- Sorts file paths before parsing to keep output deterministic.

Machine-readable symbol object format:
```json
{
  "name": "handleRequest",
  "code": "func handleRequest(w http.ResponseWriter, r *http.Request) { ... }",
  "file_path": "src/server.go",
  "start_line": 12,
  "end_line": 45,
  "start_byte": 150,
  "end_byte": 890,
  "symbol_type": "function"
}
```

## How it Works

`symgrep` leverages Tree-sitter's query language to find specific nodes in the AST. For example, in Go, it uses:
```query
[
    (function_declaration name: (identifier) @name)
    (method_declaration name: (field_identifier) @name)
    (type_declaration (type_spec name: (type_identifier) @name)) @symbol
] @symbol
```
This ensures that it only matches the *definition* of the symbol, not every time the symbol name appears in a comment or a variable call.

## Development

### Running Tests
```bash
make test
```

### Adding Languages
To add a new language:
1. Import the Tree-sitter grammar in `parser/router.go`.
2. Map the file extension to the grammar in `GetLanguage`.
3. Add the appropriate Tree-sitter query to `queries` in `parser/engine.go`.

## License
MIT
