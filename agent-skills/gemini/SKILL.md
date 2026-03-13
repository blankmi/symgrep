---
name: symgrep
description: Precision code symbol extraction and discovery using Tree-sitter. Use when you need to list functions, classes, or structs in a file or extract the exact source code of a specific symbol for surgical reading and editing. Supports Go, Python, Java, JS/TS, Rust, and C++.
---

# Symgrep Skill

## Instructions

You have access to a local binary called `bin/symgrep` (or `./bin/symgrep`) which allows you to list and extract precise code symbols from source files using Tree-sitter.

Whenever a user asks you to read or analyze a specific function or class, you should PREFER using `symgrep` over standard file reading or generic `grep`, as it will save context window tokens by returning only the exact bytes of the requested symbol.

### Workflow

1. **Discover**: If you don't know the exact symbol name, run `bin/symgrep list -f <path>` first to discover what's defined.
2. **Extract**: Run `bin/symgrep extract -f <path> -s <symbol_name> --format=json` to get the exact code and location.
3. **Use**: Use this information for analysis or to formulate precise `replace` tool calls using the line numbers.

### Commands

#### List symbols in a file or directory
```bash
./bin/symgrep list -f <file_or_dir_path>
./bin/symgrep list -f <file_or_dir_path> --format=json
./bin/symgrep list -f <file_or_dir_path> -t function
```

#### Extract specific symbol source
```bash
./bin/symgrep extract -f <file_or_dir_path> -s <symbol_name> --format=json
```

### Notes
- **Languages**: Go, Python, Java, JS/TS, Rust, C++.
- **Types**: function, method, class, struct, interface, enum, trait, impl (use `-t`).
- **Output**: JSON includes `name`, `code`, `file_path`, `start_line`, `end_line`, `start_byte`, `end_byte`, `symbol_type`.
- **Recursion**: `list` and `extract` in directory mode recurse and return sorted results.
