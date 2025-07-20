#!/bin/bash
# Go formatting hook for Claude Code
# This script formats Go files after they are written or modified

set -e

# Get the file path from stdin (Claude Code passes JSON)
if [ -t 0 ]; then
    # If no stdin, exit successfully (not called as a hook)
    exit 0
fi

# Read the JSON input
json_input=$(cat)

# Extract file path - try different field names
# Claude Code might use 'path' or 'file_path'
file_path=$(echo "$json_input" | sed -n 's/.*"path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

# If no file path found, try alternative JSON fields
if [ -z "$file_path" ]; then
    file_path=$(echo "$json_input" | sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
fi

# Check if it's a Go file
if [[ ! "$file_path" =~ \.go$ ]]; then
    # Not a Go file, exit successfully
    exit 0
fi

# Check if file exists
if [ ! -f "$file_path" ]; then
    # File doesn't exist yet (might be a Write operation), exit successfully
    exit 0
fi

echo "Formatting Go file: $file_path" >&2

# Run gofmt to format the file
if command -v gofmt >/dev/null 2>&1; then
    gofmt -w "$file_path"
else
    echo "Warning: gofmt not found, skipping formatting" >&2
fi

# Run goimports to fix imports (if available)
if command -v goimports >/dev/null 2>&1; then
    goimports -w -local "github.com/agentstation/pocket" "$file_path"
elif command -v gofmt >/dev/null 2>&1; then
    # If goimports is not available, at least run gofmt
    gofmt -w "$file_path"
fi

# Add periods to Go comments for godot compliance
# This sed command adds a period to comments that don't end with punctuation
sed -i.bak -E '
    # Match single-line comments that dont end with punctuation
    s|^([[:space:]]*)//([[:space:]]+[A-Z][^.!?:;,\n]*[a-zA-Z0-9\)"])$|\1//\2.|g
    # Match comment lines that start with a capital letter and dont end with punctuation
    s|^([[:space:]]*)//([[:space:]]+[A-Z][^.!?:;,\n]*[a-zA-Z0-9\)"])([[:space:]]*)$|\1//\2.\3|g
' "$file_path" && rm -f "${file_path}.bak"

# Run specific golangci-lint fixers if available
if command -v golangci-lint >/dev/null 2>&1; then
    # Run only specific linters that support auto-fix
    golangci-lint run --fix --disable-all \
        --enable gofmt \
        --enable goimports \
        --enable godot \
        "$file_path" 2>/dev/null || true
fi

echo "Formatting complete for: $file_path" >&2

# Exit successfully
exit 0