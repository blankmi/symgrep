# symgrep

`symgrep` is a high-precision code symbol extraction tool built for LLM agents and developers. Unlike standard `grep` which operates on text, `symgrep` uses **Tree-sitter** to parse source code into an Abstract Syntax Tree (AST), allowing it to extract exact functions, classes, methods, and structs with zero noise.

> **Benchmark note:** A/B testing across 88 agent invocations on codebases up to 250K LOC (Haiku 4.5, Sonnet 4.6) showed that current models already use Grep efficiently for symbol *search* — instructing agents to use symgrep for finding classes or listing symbols was 75% more expensive than plain Grep. 
> Where symgrep provides real value is **extraction**: pulling a single method from a large file (200+ lines) avoids reading hundreds of irrelevant lines. Position symgrep as a read-reduction tool, not a search replacement.

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
  - XML / SVG
  - HTML
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

### Adding to PATH

To use `symgrep` from any directory (and allow agents to invoke it without a relative path), add the binary to a directory in your `PATH`:

```bash
# Option 1: Copy the binary to a standard location
cp ./bin/symgrep /usr/local/bin/

# Option 2: Or add symgrep's bin directory to your PATH
export PATH="$PATH:/path/to/symgrep/bin"
# Add the line above to your ~/.zshrc or ~/.bashrc to make it permanent
```

Verify it works:
```bash
symgrep --help
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

### Ensuring agents actually use `symgrep`

Installing the skill alone is not enough — agents will default to built-in tools (grep, glob, file reads) unless you either:
1. add an explicit instruction to the agent's config, or
2. directly tell the agent in your prompt to use `symgrep`.

To make an agent reliably use `symgrep`, add the following instruction to the agent's config file in your project root:

| Agent       | Config file                          |
|-------------|--------------------------------------|
| Claude Code | `CLAUDE.md`                          |
| Gemini      | `GEMINI.md`                          |
| Codex       | `AGENTS.md` or `AGENTS.override.md`  |

```markdown
Use `symgrep extract` to read specific symbols from large files (200+ lines) instead of reading the entire file.
- Use `Grep` for all search: finding identifiers, patterns, references, class names.
- Use `symgrep extract -f <file> -s <symbol>` when you already found the file via Grep and need one method/class from a large file.
- Use `Read` when the file is short (<200 lines) or you need the full file context.
```

> [!NOTE]
> Skills and referenced files are passive — agents may not follow them reliably. Instructions placed directly in the agent's config file are loaded into the agent's context automatically and have the strongest influence on tool selection behavior.

#### Quick verification
Ask the agent something like:
- Where is handleRequest defined?
- Show me the UserService.login method.
- Find the Config struct used by the CLI.

### Use `symgrep` together with `codesight`

Combine [codesight](https://github.com/blankmi/codesight) (`cs search`) for semantic discovery with `symgrep extract` for surgical code reading. Add to your agent's project config (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`):

```markdown
# Tool Selection

- Search → Grep. Always start here for text, identifiers, patterns, class names.
- Understand → cs search "<query>" via Bash. Use for conceptual questions when you don't know which files matter.
- Extract → symgrep extract -f <file> -s <symbol> via Bash. Use instead of Read when you need one symbol from a file >200 lines.
- Find files → Glob.

Do NOT use cs search for exact-match lookups.
Do NOT read 5+ files to understand a feature — cs search ranks them for you.
```

> **Why this works:** Benchmarking on a 250K LOC codebase showed that `cs search` saves 14.5% on conceptual queries by surfacing relevant files from the semantic index, while Grep handles all pattern-based search optimally. `symgrep extract` is reserved for its one strength: reading less from large files. Verbose instructions with more rules performed worse — 7 lines outperformed 29 lines by 15 percentage points.

### Allowlisting `symgrep` for autonomous use

By default, coding agents require user approval before running shell commands. To let an agent use `symgrep` without prompting each time, add it to the agent's permission allowlist.

**Claude Code** — add to `.claude/settings.json` (project-level) or `~/.claude/settings.json` (global):

```json
{
  "permissions": {
    "allow": [
      "Bash(symgrep *)"
    ]
  }
}
```

See the [Claude Code permissions docs](https://docs.anthropic.com/en/docs/claude-code/settings#permissions) for more details on permission rules and scoping.

**Codex** — add to `~/.codex/rules/default.rules` (or project-level `.codex/rules/*.rules`):

```python
# Allow direct symgrep invocations outside the sandbox without approval prompts.
prefix_rule(
    pattern = ["symgrep"],
    decision = "allow",
)
```

See the [Codex rules docs](https://developers.openai.com/codex/rules) for rule syntax and scope details.

**Gemini CLI** — add to `.gemini/policies/symgrep.toml` (project-level) or `~/.gemini/policies/symgrep.toml` (global):

```toml
# Allow skill activation without confirmation
[[rule]]
toolName = "activate_skill"
decision = "allow"
priority = 100

# Allow symgrep command execution without confirmation
[[rule]]
toolName = "run_shell_command"
commandPrefix = "symgrep"
decision = "allow"
priority = 100
```

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
