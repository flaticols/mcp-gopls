# MCP-GOPLS

> **⚠️ WARNING: EARLY DEVELOPMENT STAGE**  
> This project is still in active development and **NOT READY FOR PRODUCTION USE**.  
> APIs are unstable and may change without notice.

## Overview

MCP-GOPLS is a wrapper around the Go language server (gopls) designed to provide language intelligence features for Go code to Claude Desktop. It enables features like code completion, hover information, definition lookup, and more.

## Features

- Code completion
- Go to definition
- Find references
- Hover information
- Code formatting
- Signature help
- Symbol search
- Code actions and quick fixes

## Status

This project is currently under active development. Many features are incomplete or may have bugs. Use at your own risk.

## Requirements

- Go 1.24 or higher
- `gopls` installed and available in your PATH

## Installation

```bash
# Not yet available - project in development
go install github.com/flaticols/mcp-gopls/cmd/mcpgopls@latest
```

## Development

```bash
# Clone the repository
git clone https://github.com/flaticols/mcp-gopls.git
cd mcp-gopls

# Build
go build -v ./cmd/mcpgopls

# Run tests
go test ./...
```

## Architecture

MCP-GOPLS consists of:

1. A wrapper around the gopls Language Server Protocol (LSP) implementation
2. A high-level API for Claude Desktop to interact with the language server

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**NOTE**: This is an early prototype and functionality may be limited or unstable.