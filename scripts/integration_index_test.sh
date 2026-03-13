#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/symgrep-integration.XXXXXX")"
BIN_PATH="$WORK_DIR/symgrep"
INDEX_DIR="$WORK_DIR/index"
INDEX_FILE="$WORK_DIR/symbol-index.json"

cleanup() {
	rm -rf "$WORK_DIR"
}
trap cleanup EXIT

log() {
	printf '[integration] %s\n' "$1"
}

fail() {
	printf '[integration][FAIL] %s\n' "$1" >&2
	exit 1
}

assert_contains() {
	local needle="$1"
	local target_file="$2"
	if ! grep -Fq "$needle" "$target_file"; then
		fail "Expected to find '$needle' in $target_file"
	fi
}

index_symbol() {
	local source_file="$1"
	local symbol="$2"
	local symbol_type="$3"
	local output_file="$4"
	local -a args

	args=(extract -f "$source_file" -s "$symbol" --format=json)
	if [[ -n "$symbol_type" ]]; then
		args+=(-t "$symbol_type")
	fi

	(
		cd "$ROOT_DIR"
		"$BIN_PATH" "${args[@]}" >"$output_file"
	)
}

mkdir -p "$INDEX_DIR"

log "Building symgrep"
(
	cd "$ROOT_DIR"
	CGO_ENABLED=1 go build -o "$BIN_PATH" ./cmd/symgrep
)

log "Indexing selected symbols from current repository"
index_symbol "cmd/symgrep/main.go" "main" "function" "$INDEX_DIR/main.json"
index_symbol "cmd/root.go" "Execute" "function" "$INDEX_DIR/execute.json"
index_symbol "cmd/extract.go" "runExtract" "function" "$INDEX_DIR/run_extract.json"
index_symbol "parser/router.go" "GetLanguage" "function" "$INDEX_DIR/get_language.json"
index_symbol "parser/engine.go" "ExtractSymbol" "function" "$INDEX_DIR/extract_symbol.json"
index_symbol "parser/engine.go" "SymbolInfo" "struct" "$INDEX_DIR/symbol_info.json"

log "Creating aggregated symbol index"
first=1
{
	printf '[\n'
	for json_file in "$INDEX_DIR"/*.json; do
		if [[ $first -eq 0 ]]; then
			printf ',\n'
		fi
		cat "$json_file"
		first=0
	done
	printf '\n]\n'
} >"$INDEX_FILE"

log "Running assertions against generated index"
for json_file in "$INDEX_DIR"/*.json; do
	assert_contains '"code": ' "$json_file"
	assert_contains '"file_path": "' "$json_file"
	assert_contains '"start_line": ' "$json_file"
	assert_contains '"end_line": ' "$json_file"
	assert_contains '"symbol_type": "' "$json_file"
done

assert_contains '"file_path": "cmd/symgrep/main.go"' "$INDEX_DIR/main.json"
assert_contains '"code": "func main() {' "$INDEX_DIR/main.json"
assert_contains '"symbol_type": "function"' "$INDEX_DIR/main.json"

assert_contains '"file_path": "cmd/root.go"' "$INDEX_DIR/execute.json"
assert_contains '"code": "func Execute() error {' "$INDEX_DIR/execute.json"

assert_contains '"file_path": "parser/engine.go"' "$INDEX_DIR/symbol_info.json"
assert_contains '"code": "type SymbolInfo struct {' "$INDEX_DIR/symbol_info.json"
assert_contains '"symbol_type": "struct"' "$INDEX_DIR/symbol_info.json"

assert_contains '"file_path": "parser/router.go"' "$INDEX_FILE"
assert_contains '"file_path": "cmd/extract.go"' "$INDEX_FILE"

log "Verifying error path for unknown symbol"
if (
	cd "$ROOT_DIR"
	"$BIN_PATH" extract -f parser/engine.go -s DoesNotExist >/dev/null 2>"$WORK_DIR/missing.err"
); then
	fail "Expected extraction of missing symbol to fail"
fi
assert_contains "not found" "$WORK_DIR/missing.err"

log "Verifying list command (raw)"
list_raw_output="$(
	cd "$ROOT_DIR"
	"$BIN_PATH" list -f parser/engine.go
)"
case "$list_raw_output" in
*"function"*"ExtractSymbol"*)  ;;
*)
	fail "Expected list raw output to contain ExtractSymbol function"
	;;
esac

log "Verifying list command (json)"
list_json_output="$(
	cd "$ROOT_DIR"
	"$BIN_PATH" list -f parser/engine.go --format=json
)"
case "$list_json_output" in
*'"name":'*)  ;;
*)
	fail "Expected list JSON output to contain name field"
	;;
esac

log "Verifying list command with type filter"
list_filtered_output="$(
	cd "$ROOT_DIR"
	"$BIN_PATH" list -f parser/engine.go -t function
)"
case "$list_filtered_output" in
*"struct"*)
	fail "Expected filtered list to not contain struct symbols"
	;;
*"function"*)  ;;
*)
	fail "Expected filtered list to contain function symbols"
	;;
esac

log "Verifying directory list command (raw)"
list_dir_raw_output="$(
	cd "$ROOT_DIR"
	"$BIN_PATH" list -f parser/
)"
case "$list_dir_raw_output" in
*"parser/engine.go"*"ExtractSymbol"*) ;;
*)
	fail "Expected directory list raw output to contain parser symbols"
	;;
esac

log "Verifying directory extract command (json)"
extract_dir_json_output="$(
	cd "$ROOT_DIR"
	"$BIN_PATH" extract -f parser/ -s GetLanguage --format=json
)"
case "$extract_dir_json_output" in
*'"file_path": "parser/router.go"'*) ;;
*)
	fail "Expected directory extract JSON output to include parser/router.go"
	;;
esac

log "Verifying directory list command with language filter"
MIXED_DIR="$WORK_DIR/mixed"
mkdir -p "$MIXED_DIR"
cat >"$MIXED_DIR/main.go" <<'EOF'
package main

func GoOnly() {}
EOF
cat >"$MIXED_DIR/helper.py" <<'EOF'
def py_only():
    return 1
EOF

list_dir_filtered_output="$(
	cd "$ROOT_DIR"
	"$BIN_PATH" list -f "$MIXED_DIR" -l golang
)"
case "$list_dir_filtered_output" in
*"GoOnly"*) ;;
*)
	fail "Expected language-filtered directory list to contain GoOnly"
	;;
esac
case "$list_dir_filtered_output" in
*"py_only"*)
	fail "Expected language-filtered directory list to exclude Python symbols"
	;;
*) ;;
esac

log "Verifying directory list error when no supported files exist"
NO_SUPPORTED_DIR="$WORK_DIR/no-supported"
mkdir -p "$NO_SUPPORTED_DIR"
echo "notes" >"$NO_SUPPORTED_DIR/readme.txt"
if (
	cd "$ROOT_DIR"
	"$BIN_PATH" list -f "$NO_SUPPORTED_DIR" >/dev/null 2>"$WORK_DIR/no_supported.err"
); then
	fail "Expected directory list without supported files to fail"
fi
assert_contains "no supported files found" "$WORK_DIR/no_supported.err"

log "Verifying directory list error for unsupported language filter"
if (
	cd "$ROOT_DIR"
	"$BIN_PATH" list -f parser/ -l ruby >/dev/null 2>"$WORK_DIR/unsupported_lang.err"
); then
	fail "Expected unsupported language filter to fail"
fi
assert_contains "unsupported language filter" "$WORK_DIR/unsupported_lang.err"

log "Verifying default raw output mode"
raw_output="$(
	cd "$ROOT_DIR"
	"$BIN_PATH" extract -f cmd/root.go -s Execute
)"
case "$raw_output" in
*"func Execute() error"*) ;;
*)
	fail "Expected raw output for Execute symbol"
	;;
esac

log "Integration test passed"
