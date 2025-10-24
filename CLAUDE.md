# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Omniverse is a command-line tool for string substitution across multiple files. It reads a source directory with a manifest file (`.alterverse.yml`), and creates a modified copy in a destination directory using that destination's manifest. The tool performs simple, deterministic string substitution without complex templating logic.

## Build and Test Commands

```bash
# Build the project
go build

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -run TestName

# Run tests with coverage
go test -cover ./...

# Build for release (uses goreleaser)
goreleaser build --snapshot --clean
```

## Architecture

### Core Components

The codebase follows a modular architecture with distinct responsibilities:

**Alterverse** (`alterverse.go`)
- Represents a directory with its manifest (`.alterverse.yml` file)
- Validates manifests to ensure no duplicate values (which would make bidirectional conversion impossible)
- Uses a `Syncer` to read/write files from/to the filesystem
- Exposes `Files()` and `WriteFiles()` methods for file operations

**Interverse** (`interverse.go`)
- Handles the conversion logic between two Alterverses
- Builds a lookup table from source and destination manifests
- Implements two deduction methods:
  - `Deduce()`: Fast string substitution (may not be reversible)
  - `DeduceStrict()`: Slower but guarantees bidirectional conversion
- Uses a tokenizer-based approach to perform string replacements
- Sorts lookup table by value length (longest first) to prevent substring conflicts

**Syncer** (`syncer.go`)
- Low-level file I/O abstraction
- Reads and writes files relative to a base directory
- Handles file ignore patterns via regex (default: `^.*[\\/]\..*|^\..*` ignores hidden files)
- Manages file deletion for obsolete files during sync operations
- Properly handles file truncation when writing shorter content

**Tokenizer** (in `interverse.go`)
- Splits byte slices into tokens for string replacement
- Two token types: `byteToken` (unchanged data) and `switchToken` (replaced data)
- Allows verification that replacements can be reversed

**CLI** (`cli.go`)
- Built with `spf13/cobra`
- Main commands:
  - `deduce`: Convert files from one alterverse to another
  - `contexts` (hidden): Debug tool to find contexts where manifest values appear
  - `version`: Display version information
- Default ignore pattern for hidden files/directories

### Key Data Flow

1. `NewAlterverse()` loads source and destination directories with their manifests
2. `NewInterverse()` creates lookup table from both manifests, validates matching keys
3. `DeduceStrict()` performs string substitution using the tokenizer
4. Verification ensures the conversion is reversible
5. `WriteFiles()` syncs changes to the destination, deleting obsolete files

## Testing

Tests are table-based and use the `testdata/` directory for fixtures. Golden files are used where appropriate. Test files include:
- `alterverse_test.go`: Manifest validation and alterverse creation
- `interverse_test.go`: Deduction logic and tokenization
- `syncer_test.go`: File I/O operations
- `cli_test.go`: CLI command parsing
- `diff_test.go`: Diff generation utilities

## Important Implementation Details

### String Replacement Strategy
The lookup table is sorted by value length (longest first, via `sort.Reverse()`) to ensure longer strings are replaced before shorter ones that might be substrings. This prevents incorrect replacements.

### Bidirectional Guarantee
`DeduceStrict()` verifies that:
1. The destination alterverse's values don't already exist in source files
2. Converting from A→B and then B→A produces identical content

### File Ignore Patterns
The default ignore pattern (`^.*[\\/]\..*|^\..*`) excludes:
- Hidden files and directories (starting with `.`)
- The `.alterverse.yml` manifest itself is excluded from processing

### Manifest Structure
Manifests are YAML files with a simple key-value structure:
```yaml
manifest:
  key1: value1
  key2: value2
```

Keys must be identical across source and destination manifests. Values must be unique within each manifest (no duplicate values allowed).
