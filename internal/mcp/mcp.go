package mcp

import (
	"fmt"
	"github.com/flaticols/mcpgopls/internal/gopls"
)

// ClaudeGoplsServer is a high-level interface for Claude Desktop to interact with gopls.
type ClaudeGoplsServer struct {
	wrapper *gopls.GoplsWrapper
	config  *gopls.Config
	rootDir string
}

// NewClaudeGoplsServer creates a new Claude Gopls server with the given configuration.
func NewClaudeGoplsServer(config *gopls.Config, rootDir string) (*ClaudeGoplsServer, error) {
	if config == nil {
		config = gopls.NewConfig()
	}

	wrapper, err := gopls.NewGoplsWrapper()
	if err != nil {
		return nil, err
	}

	server := &ClaudeGoplsServer{
		wrapper: wrapper,
		config:  config,
		rootDir: rootDir,
	}

	return server, nil
}

// Start starts the gopls server and initializes it.
func (s *ClaudeGoplsServer) Start() error {
	err := s.wrapper.Start()
	if err != nil {
		return err
	}

	_, err = s.wrapper.Initialize(s.rootDir)
	if err != nil {
		s.Stop()
		return err
	}

	return nil
}

// Stop stops the gopls server.
func (s *ClaudeGoplsServer) Stop() error {
	// Try to shut down gracefully
	_ = s.wrapper.Shutdown()

	// Force stop if needed
	return s.wrapper.Stop()
}

// OpenFile notifies gopls that a file has been opened.
func (s *ClaudeGoplsServer) OpenFile(filepath string, content string) error {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.DidOpen(uri, content, "go")
}

// GetCompletions gets completion suggestions at the given position.
func (s *ClaudeGoplsServer) GetCompletions(filepath string, line, character int) ([]gopls.CompletionItem, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	result, err := s.wrapper.Completion(uri, line, character)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// GetDefinition gets the definition of a symbol at the given position.
func (s *ClaudeGoplsServer) GetDefinition(filepath string, line, character int) ([]gopls.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.Definition(uri, line, character)
}

// GetHover gets hover information at the given position.
func (s *ClaudeGoplsServer) GetHover(filepath string, line, character int) (*gopls.Hover, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.Hover(uri, line, character)
}

// FormatFile formats a file.
func (s *ClaudeGoplsServer) FormatFile(filepath string, content string) (string, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	// Open the file if it's not already open
	err := s.wrapper.DidOpen(uri, content, "go")
	if err != nil {
		return "", err
	}

	// Get formatting edits
	edits, err := s.wrapper.Format(uri)
	if err != nil {
		return "", err
	}

	// Apply edits to content
	// This is a simplified version that assumes edits are sorted
	result := content
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		startOffset := offsetForPosition(content, edit.Range.Start)
		endOffset := offsetForPosition(content, edit.Range.End)

		result = result[:startOffset] + edit.NewText + result[endOffset:]
	}

	return result, nil
}

// offsetForPosition converts a Position to a byte offset in text.
// Note: This is a simplified version that doesn't handle UTF-8 correctly.
func offsetForPosition(text string, pos gopls.Position) int {
	lines := []int{0}
	for i, c := range text {
		if c == '\n' {
			lines = append(lines, i+1)
		}
	}

	if int(pos.Line) >= len(lines) {
		return len(text)
	}

	lineStart := lines[pos.Line]
	offset := lineStart

	// Count characters
	charCount := 0
	for i := lineStart; i < len(text) && i < len(text); i++ {
		if charCount >= int(pos.Character) {
			break
		}
		if text[i] != '\r' { // Skip carriage returns
			charCount++
			offset = i + 1
		}
	}

	return offset
}

// GetImplementations gets the implementations of a symbol at the given position.
func (s *ClaudeGoplsServer) GetImplementations(filepath string, line, character int) ([]gopls.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.Implementation(uri, line, character)
}

// GetReferences gets the references to a symbol at the given position.
func (s *ClaudeGoplsServer) GetReferences(filepath string, line, character int) ([]gopls.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.References(uri, line, character, true)
}

// GetRenameEdits gets the edits needed to rename a symbol at the given position.
func (s *ClaudeGoplsServer) GetRenameEdits(filepath string, line, character int, newName string) (*gopls.WorkspaceEdit, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.Rename(uri, line, character, newName)
}

// GetCodeActions gets the available code actions at the given range.
func (s *ClaudeGoplsServer) GetCodeActions(filepath string, startLine, startChar, endLine, endChar int) ([]gopls.CodeAction, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.CodeAction(uri, startLine, startChar, endLine, endChar, nil, nil)
}

// UpdateFile notifies gopls that a file has been changed.
func (s *ClaudeGoplsServer) UpdateFile(filepath string, content string, version int) error {
	uri := fmt.Sprintf("file://%s", filepath)
	changes := []gopls.TextDocumentContentChangeEvent{
		{
			Text: content,
		},
	}
	return s.wrapper.DidChange(uri, version, changes)
}

// CloseFile notifies gopls that a file has been closed.
func (s *ClaudeGoplsServer) CloseFile(filepath string) error {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.DidClose(uri)
}

// SaveFile notifies gopls that a file has been saved.
func (s *ClaudeGoplsServer) SaveFile(filepath string, content *string) error {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.DidSave(uri, content)
}

// ExecuteGoplsCommand executes a gopls command.
func (s *ClaudeGoplsServer) ExecuteGoplsCommand(command string, args []interface{}) (interface{}, error) {
	return s.wrapper.ExecuteCommand(command, args)
}

// GetSignatureHelp gets signature help at the given position.
func (s *ClaudeGoplsServer) GetSignatureHelp(filepath string, line, character int) (*gopls.SignatureHelp, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.SignatureHelp(uri, line, character)
}

// GetDocumentSymbols gets all symbols in a document.
func (s *ClaudeGoplsServer) GetDocumentSymbols(filepath string) ([]gopls.DocumentSymbol, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.DocumentSymbol(uri)
}

// FindWorkspaceSymbols searches for symbols in the workspace.
func (s *ClaudeGoplsServer) FindWorkspaceSymbols(query string) ([]gopls.SymbolInformation, error) {
	return s.wrapper.WorkspaceSymbol(query)
}

// GetTypeDefinition gets the type definition of a symbol at the given position.
func (s *ClaudeGoplsServer) GetTypeDefinition(filepath string, line, character int) ([]gopls.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)
	return s.wrapper.TypeDefinition(uri, line, character)
}
