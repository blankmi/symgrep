---
name: symgrep
description: Semantic code search and symbol extraction using the local `symgrep` binary (Tree-sitter AST). Use when you need to discover symbols in a file/directory or extract a specific function/method/class/struct for precise reading or editing without loading full files.
---

# Symgrep

Use `symgrep` for symbol-level navigation and targeted code extraction.

Prefer `./symgrep` from the repository root. If unavailable in the current working directory, use `symgrep` from `PATH`.

## When To Use

- You need to discover what symbols exist in a code file.
- You need to search a codebase directory recursively for symbols.
- You need exact code for one symbol (function, method, class, struct).
- You want line/byte ranges for precise edits.
- The file is large and only symbol-level context should be read.

## Workflow

1. Start with `list` on a file or directory to discover candidate symbols.
2. Use `-t` and `-l` to narrow results when the match set is large.
3. Run `extract` on the selected symbol from the file or directory target.
4. Use JSON output when line/byte offsets or machine parsing are needed.

## Commands

```bash
# List all symbols in a file (tab-separated: type, name, line range)
./symgrep list -f path/to/file.go

# List as JSON (includes full source for each symbol)
./symgrep list -f path/to/file.go --format=json

# List only one symbol type
./symgrep list -f path/to/file.go -t function

# List recursively across a directory
./symgrep list -f path/to/dir/

# Directory search narrowed by type and language
./symgrep list -f path/to/dir/ -t function -l go
```

```bash
# Extract symbol source (raw code)
./symgrep extract -f path/to/file.go -s FunctionName

# Extract symbol as JSON with locations
./symgrep extract -f path/to/file.go -s FunctionName --format=json

# Extract recursively across a directory
./symgrep extract -f path/to/dir/ -s FunctionName --format=json

# Directory search in raw mode returns each match grouped by file
./symgrep extract -f path/to/dir/ -s FunctionName
```

## Output Details

- Exit code: `0` success, `1` error.
- Errors are written to `stderr` only.
- `list` raw format:
  - file mode: `type\tname\tLstart-Lend`
  - directory mode: `path\ttype\tname\tLstart-Lend`
- `list` JSON format: array of objects with:
  - `name`, `code`, `file_path`, `start_line`, `end_line`, `start_byte`, `end_byte`, `symbol_type`
- `extract` JSON format:
  - file mode: one object
  - directory mode: array of matches
- `extract` raw format:
  - file mode: raw symbol source
  - directory mode: each match is prefixed with `=== file: <path> ===`

## Directory Traversal Rules

- Recursively scans supported files under the target directory.
- Skips common generated/vendor/tooling folders (for example: `.git`, `node_modules`, `vendor`, `__pycache__`, `.idea`, `.vscode`).
- Uses deterministic path ordering to keep output stable across runs.

## Supported Languages

- Go (`.go`)
- Python (`.py`)
- JavaScript/TypeScript (`.js`, `.jsx`, `.ts`, `.tsx`)
- Java (`.java`)
- Rust (`.rs`)
- C++ (`.cpp`, `.cc`, `.h`)

## Operating Rule

Always use `symgrep list` before `symgrep extract` to keep context lean and avoid symbol-name guesswork.
