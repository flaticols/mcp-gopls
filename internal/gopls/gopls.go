// Package gopls provides a wrapper for gopls to be used within Claude Desktop.
package gopls

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// LSP Protocol types
// These are minimal implementations of the types needed for LSP

// DocumentURI represents a URI for a document
type DocumentURI string

// Position represents a position in a document
type Position struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

// Range represents a range in a document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a document
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// TextDocumentIdentifier identifies a document
type TextDocumentIdentifier struct {
	URI DocumentURI `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a document with a version
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentItem represents a text document
type TextDocumentItem struct {
	URI        DocumentURI `json:"uri"`
	LanguageID string      `json:"languageId"`
	Version    int         `json:"version"`
	Text       string      `json:"text"`
}

// TextDocumentContentChangeEvent represents a change to a document
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength uint32 `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// TextEdit represents an edit to a document
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// WorkspaceEdit represents edits to multiple documents
type WorkspaceEdit struct {
	Changes         map[DocumentURI][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []TextDocumentEdit         `json:"documentChanges,omitempty"`
}

// TextDocumentEdit represents edits to a document
type TextDocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                      `json:"edits"`
}

// Diagnostic represents a diagnostic for a document
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity uint32 `json:"severity,omitempty"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// Command represents a command
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// CodeAction represents a code action
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        CodeActionKind `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	IsPreferred bool           `json:"isPreferred,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
}

// CodeActionKind represents the kind of a code action
type CodeActionKind string

// CompletionItem represents a completion item
type CompletionItem struct {
	Label         string      `json:"label"`
	Kind          uint32      `json:"kind,omitempty"`
	Detail        string      `json:"detail,omitempty"`
	Documentation string      `json:"documentation,omitempty"`
	InsertText    string      `json:"insertText,omitempty"`
	TextEdit      *TextEdit   `json:"textEdit,omitempty"`
	Data          interface{} `json:"data,omitempty"`
}

// CompletionList represents a list of completion items
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// SignatureHelp represents signature help
type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature uint32                 `json:"activeSignature,omitempty"`
	ActiveParameter uint32                 `json:"activeParameter,omitempty"`
}

// SignatureInformation represents information about a signature
type SignatureInformation struct {
	Label           string                 `json:"label"`
	Documentation   string                 `json:"documentation,omitempty"`
	Parameters      []ParameterInformation `json:"parameters,omitempty"`
	ActiveParameter uint32                 `json:"activeParameter,omitempty"`
}

// ParameterInformation represents information about a parameter
type ParameterInformation struct {
	Label         string `json:"label"`
	Documentation string `json:"documentation,omitempty"`
}

// Hover represents hover information
type Hover struct {
	Contents interface{} `json:"contents"`
	Range    *Range      `json:"range,omitempty"`
}

// MarkupContent represents markup content
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// DocumentSymbol represents a symbol in a document
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           uint32           `json:"kind"`
	Deprecated     bool             `json:"deprecated,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolInformation represents information about a symbol
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          uint32   `json:"kind"`
	Deprecated    bool     `json:"deprecated,omitempty"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

// WorkspaceFolder represents a workspace folder
type WorkspaceFolder struct {
	URI  DocumentURI `json:"uri"`
	Name string      `json:"name"`
}

// InitializeParams represents parameters for initialize request
type InitializeParams struct {
	ProcessID        int                `json:"processId,omitempty"`
	RootURI          DocumentURI        `json:"rootUri,omitempty"`
	WorkspaceFolders []WorkspaceFolder  `json:"workspaceFolders,omitempty"`
	Capabilities     ClientCapabilities `json:"capabilities"`
}

// TextDocumentClientCapabilities represents client capabilities for text documents
type TextDocumentClientCapabilities struct {
	Completion     *CompletionClientCapabilities         `json:"completion,omitempty"`
	Hover          *HoverClientCapabilities              `json:"hover,omitempty"`
	SignatureHelp  *SignatureHelpClientCapabilities      `json:"signatureHelp,omitempty"`
	References     *ReferenceClientCapabilities          `json:"references,omitempty"`
	Definition     *DefinitionClientCapabilities         `json:"definition,omitempty"`
	Implementation *ImplementationClientCapabilities     `json:"implementation,omitempty"`
	TypeDefinition *TypeDefinitionClientCapabilities     `json:"typeDefinition,omitempty"`
	DocumentSymbol *DocumentSymbolClientCapabilities     `json:"documentSymbol,omitempty"`
	CodeAction     *CodeActionClientCapabilities         `json:"codeAction,omitempty"`
	Formatting     *DocumentFormattingClientCapabilities `json:"formatting,omitempty"`
	Rename         *RenameClientCapabilities             `json:"rename,omitempty"`
}

// WorkspaceClientCapabilities represents client capabilities for workspace
type WorkspaceClientCapabilities struct {
	Symbol         *WorkspaceSymbolClientCapabilities `json:"symbol,omitempty"`
	ExecuteCommand *ExecuteCommandClientCapabilities  `json:"executeCommand,omitempty"`
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

// CompletionClientCapabilities represents client capabilities for completion
type CompletionClientCapabilities struct {
	CompletionItem *CompletionItemCapabilities `json:"completionItem,omitempty"`
}

// CompletionItemCapabilities represents client capabilities for completion items
type CompletionItemCapabilities struct {
	SnippetSupport bool `json:"snippetSupport,omitempty"`
}

// HoverClientCapabilities represents client capabilities for hover
type HoverClientCapabilities struct{}

// SignatureHelpClientCapabilities represents client capabilities for signature help
type SignatureHelpClientCapabilities struct {
	SignatureInformation *SignatureInformationCapabilities `json:"signatureInformation,omitempty"`
}

// SignatureInformationCapabilities represents client capabilities for signature information
type SignatureInformationCapabilities struct {
	ParameterInformation *ParameterInformationCapabilities `json:"parameterInformation,omitempty"`
}

// ParameterInformationCapabilities represents client capabilities for parameter information
type ParameterInformationCapabilities struct {
	LabelOffsetSupport bool `json:"labelOffsetSupport,omitempty"`
}

// ReferenceClientCapabilities represents client capabilities for references
type ReferenceClientCapabilities struct{}

// DefinitionClientCapabilities represents client capabilities for definitions
type DefinitionClientCapabilities struct{}

// ImplementationClientCapabilities represents client capabilities for implementations
type ImplementationClientCapabilities struct{}

// TypeDefinitionClientCapabilities represents client capabilities for type definitions
type TypeDefinitionClientCapabilities struct{}

// DocumentSymbolClientCapabilities represents client capabilities for document symbols
type DocumentSymbolClientCapabilities struct{}

// CodeActionClientCapabilities represents client capabilities for code actions
type CodeActionClientCapabilities struct{}

// DocumentFormattingClientCapabilities represents client capabilities for document formatting
type DocumentFormattingClientCapabilities struct{}

// RenameClientCapabilities represents client capabilities for rename
type RenameClientCapabilities struct{}

// WorkspaceSymbolClientCapabilities represents client capabilities for workspace symbols
type WorkspaceSymbolClientCapabilities struct{}

// ExecuteCommandClientCapabilities represents client capabilities for execute command
type ExecuteCommandClientCapabilities struct{}

// InitializeResult represents the result of an initialize request
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	TextDocumentSync           interface{}            `json:"textDocumentSync,omitempty"`
	CompletionProvider         *CompletionOptions     `json:"completionProvider,omitempty"`
	HoverProvider              bool                   `json:"hoverProvider,omitempty"`
	SignatureHelpProvider      *SignatureHelpOptions  `json:"signatureHelpProvider,omitempty"`
	DeclarationProvider        bool                   `json:"declarationProvider,omitempty"`
	DefinitionProvider         bool                   `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider     bool                   `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider     bool                   `json:"implementationProvider,omitempty"`
	ReferencesProvider         bool                   `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider     bool                   `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider         interface{}            `json:"codeActionProvider,omitempty"`
	DocumentFormattingProvider bool                   `json:"documentFormattingProvider,omitempty"`
	RenameProvider             interface{}            `json:"renameProvider,omitempty"`
	WorkspaceSymbolProvider    bool                   `json:"workspaceSymbolProvider,omitempty"`
	ExecuteCommandProvider     *ExecuteCommandOptions `json:"executeCommandProvider,omitempty"`
}

// CompletionOptions represents options for completion
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
}

// SignatureHelpOptions represents options for signature help
type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// ExecuteCommandOptions represents options for execute command
type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

// InitializedParams represents parameters for initialized notification
type InitializedParams struct{}

// DidOpenTextDocumentParams represents parameters for textDocument/didOpen notification
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams represents parameters for textDocument/didChange notification
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams represents parameters for textDocument/didClose notification
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams represents parameters for textDocument/didSave notification
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

// TextDocumentPositionParams represents parameters for text document position requests
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionParams represents parameters for textDocument/completion request
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext represents context for completion
type CompletionContext struct {
	TriggerKind      uint32  `json:"triggerKind"`
	TriggerCharacter *string `json:"triggerCharacter,omitempty"`
}

// DefinitionParams represents parameters for textDocument/definition request
type DefinitionParams struct {
	TextDocumentPositionParams
}

// HoverParams represents parameters for textDocument/hover request
type HoverParams struct {
	TextDocumentPositionParams
}

// SignatureHelpParams represents parameters for textDocument/signatureHelp request
type SignatureHelpParams struct {
	TextDocumentPositionParams
	Context *SignatureHelpContext `json:"context,omitempty"`
}

// SignatureHelpContext represents context for signature help
type SignatureHelpContext struct {
	TriggerKind      uint32  `json:"triggerKind"`
	TriggerCharacter *string `json:"triggerCharacter,omitempty"`
	IsRetrigger      bool    `json:"isRetrigger"`
	ActiveSignature  uint32  `json:"activeSignature,omitempty"`
	ActiveParameter  uint32  `json:"activeParameter,omitempty"`
}

// ReferenceParams represents parameters for textDocument/references request
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext represents context for references
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// DocumentSymbolParams represents parameters for textDocument/documentSymbol request
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentFormattingParams represents parameters for textDocument/formatting request
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// FormattingOptions represents options for formatting
type FormattingOptions struct {
	TabSize      uint32          `json:"tabSize"`
	InsertSpaces bool            `json:"insertSpaces"`
	Properties   map[string]bool `json:"properties,omitempty"`
}

// CodeActionParams represents parameters for textDocument/codeAction request
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext represents context for code actions
type CodeActionContext struct {
	Diagnostics []Diagnostic     `json:"diagnostics"`
	Only        []CodeActionKind `json:"only,omitempty"`
}

// RenameParams represents parameters for textDocument/rename request
type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

// ImplementationParams represents parameters for textDocument/implementation request
type ImplementationParams struct {
	TextDocumentPositionParams
}

// TypeDefinitionParams represents parameters for textDocument/typeDefinition request
type TypeDefinitionParams struct {
	TextDocumentPositionParams
}

// WorkspaceSymbolParams represents parameters for workspace/symbol request
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// ExecuteCommandParams represents parameters for workspace/executeCommand request
type ExecuteCommandParams struct {
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// Message represents a JSON-RPC 2.0 message
type Message struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error
type Error struct {
	Code    int64            `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data,omitempty"`
}

// HeaderWriter writes JSON-RPC messages with Content-Length headers
type HeaderWriter struct {
	w io.Writer
}

// NewHeaderWriter creates a new HeaderWriter
func NewHeaderWriter(w io.Writer) *HeaderWriter {
	return &HeaderWriter{w: w}
}

// Write writes a message with Content-Length header
func (w *HeaderWriter) Write(data []byte) (int, error) {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := w.w.Write([]byte(header)); err != nil {
		return 0, err
	}
	return w.w.Write(data)
}

// HeaderReader reads JSON-RPC messages with Content-Length headers
type HeaderReader struct {
	r         *bufio.Reader
	remaining int
}

// NewHeaderReader creates a new HeaderReader
func NewHeaderReader(r io.Reader) *HeaderReader {
	return &HeaderReader{
		r:         bufio.NewReader(r),
		remaining: 0,
	}
}

// Read reads a message with Content-Length header
func (r *HeaderReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		// Read headers
		for {
			line, err := r.r.ReadString('\n')
			if err != nil {
				return 0, err
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				lengthStr := strings.TrimSpace(line[len("Content-Length:"):])
				length, err := strconv.Atoi(lengthStr)
				if err != nil {
					return 0, fmt.Errorf("invalid Content-Length: %v", err)
				}
				r.remaining = length
			}
		}
		if r.remaining == 0 {
			return 0, fmt.Errorf("no Content-Length header found")
		}
	}

	// Read message body
	if r.remaining <= len(p) {
		n, err := io.ReadFull(r.r, p[:r.remaining])
		r.remaining = 0
		return n, err
	}
	n, err := io.ReadFull(r.r, p)
	r.remaining -= n
	return n, err
}

// GoplsWrapper represents a wrapper for gopls to be used with Claude Desktop.
type GoplsWrapper struct {
	cmd        *exec.Cmd
	stdin      *HeaderWriter
	stdout     *HeaderReader
	stderr     io.ReadCloser
	mu         sync.Mutex
	handlers   map[string]func(interface{}) (interface{}, error)
	nextID     int64
	responseCh map[string]chan *Message
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewGoplsWrapper creates a new instance of GoplsWrapper.
func NewGoplsWrapper() (*GoplsWrapper, error) {
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		return nil, fmt.Errorf("gopls not found in PATH: %v", err)
	}

	// Default args
	args := []string{"-rpc.trace"}
	
	cmd := exec.Command(goplsPath, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	wrapper := &GoplsWrapper{
		cmd:        cmd,
		stdin:      NewHeaderWriter(stdin),
		stdout:     NewHeaderReader(stdout),
		stderr:     stderr,
		handlers:   make(map[string]func(interface{}) (interface{}, error)),
		nextID:     1,
		responseCh: make(map[string]chan *Message),
		ctx:        ctx,
		cancel:     cancel,
	}

	return wrapper, nil
}

// Start starts the gopls server.
func (w *GoplsWrapper) Start() error {
	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gopls: %v", err)
	}

	// Start a goroutine to read messages from stdout
	go w.readMessages()

	// Start a goroutine to handle stderr output
	go w.readStderr()

	return nil
}

// readMessages continuously reads messages from stdout
func (w *GoplsWrapper) readMessages() {
	decoder := json.NewDecoder(w.stdout)
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF || w.ctx.Err() != nil {
					return
				}
				log.Printf("Error decoding message: %v", err)
				continue
			}

			// Handle the message based on whether it's a request, response, or notification
			w.handleMessage(&msg)
		}
	}
}

// handleMessage processes received messages
func (w *GoplsWrapper) handleMessage(msg *Message) {
	if msg.Method != "" && msg.ID == nil {
		// It's a notification, find the handler
		w.mu.Lock()
		handler, ok := w.handlers[msg.Method]
		w.mu.Unlock()

		if ok {
			// Parse the params
			var params interface{}
			if len(msg.Params) > 0 {
				if err := json.Unmarshal(msg.Params, &params); err != nil {
					log.Printf("Error parsing notification params: %v", err)
					return
				}
			}

			// Handle the notification
			_, _ = handler(params)
		}
	} else if msg.Method != "" && msg.ID != nil {
		// It's a request, find the handler
		w.mu.Lock()
		handler, ok := w.handlers[msg.Method]
		w.mu.Unlock()

		if ok {
			// Parse the params
			var params interface{}
			if len(msg.Params) > 0 {
				if err := json.Unmarshal(msg.Params, &params); err != nil {
					log.Printf("Error parsing request params: %v", err)
					return
				}
			}

			// Handle the request
			result, err := handler(params)

			// Send the response
			if err != nil {
				if err = w.sendErrorResponse(string(msg.ID), err); err != nil {
					slog.Error("Failed to send error response: %v", slog.String("error", err.Error()))
				}
			} else {
				if err := w.sendResponse(string(msg.ID), result); err != nil {
					slog.Error("Failed to send response: %v", slog.String("error", err.Error()))
				}
			}
		} else {
			if err := w.sendErrorResponse(string(msg.ID), fmt.Errorf("method not found: %s", msg.Method)); err != nil {
				slog.Error("Failed to send error response: %v", slog.String("error", err.Error()))
			}
		}
	} else if msg.ID != nil {
		// It's a response, dispatch to the waiting channel
		idStr := string(msg.ID)

		w.mu.Lock()
		ch, ok := w.responseCh[idStr]
		w.mu.Unlock()

		if ok {
			ch <- msg
		}
	}
}

// readStderr reads stderr output from gopls
func (w *GoplsWrapper) readStderr() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			n, err := w.stderr.Read(buf)
			if err != nil {
				if err != io.EOF && w.ctx.Err() == nil {
					log.Printf("Error reading from gopls stderr: %v", err)
				}
				return
			}
			log.Printf("Gopls: %s", buf[:n])
		}
	}
}

// Stop stops the gopls server.
func (w *GoplsWrapper) Stop() error {
	w.cancel()

	if w.cmd.Process != nil {
		return w.cmd.Process.Kill()
	}
	return nil
}

// Initialize initializes the gopls server with the given root directory.
func (w *GoplsWrapper) Initialize(rootDir string) (*InitializeResult, error) {
	params := &InitializeParams{
		ProcessID: os.Getpid(),
		RootURI:   DocumentURI(fmt.Sprintf("file://%s", filepath.ToSlash(rootDir))),
		WorkspaceFolders: []WorkspaceFolder{{
			URI:  DocumentURI(fmt.Sprintf("file://%s", filepath.ToSlash(rootDir))),
			Name: filepath.Base(rootDir),
		}},
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Completion: &CompletionClientCapabilities{
					CompletionItem: &CompletionItemCapabilities{
						SnippetSupport: true,
					},
				},
				Hover: &HoverClientCapabilities{},
				SignatureHelp: &SignatureHelpClientCapabilities{
					SignatureInformation: &SignatureInformationCapabilities{
						ParameterInformation: &ParameterInformationCapabilities{
							LabelOffsetSupport: true,
						},
					},
				},
				References:     &ReferenceClientCapabilities{},
				Definition:     &DefinitionClientCapabilities{},
				Implementation: &ImplementationClientCapabilities{},
				TypeDefinition: &TypeDefinitionClientCapabilities{},
				DocumentSymbol: &DocumentSymbolClientCapabilities{},
				CodeAction:     &CodeActionClientCapabilities{},
				Formatting:     &DocumentFormattingClientCapabilities{},
				Rename:         &RenameClientCapabilities{},
			},
			Workspace: WorkspaceClientCapabilities{
				Symbol:         &WorkspaceSymbolClientCapabilities{},
				ExecuteCommand: &ExecuteCommandClientCapabilities{},
			},
		},
	}

	var result InitializeResult
	err := w.Call("initialize", params, &result)
	if err != nil {
		return nil, err
	}

	// Send initialized notification
	err = w.Notify("initialized", &InitializedParams{})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// Shutdown shuts down the gopls server.
func (w *GoplsWrapper) Shutdown() error {
	var result interface{}
	err := w.Call("shutdown", nil, &result)
	if err != nil {
		return err
	}

	return w.Notify("exit", nil)
}

// DidOpen notifies the server about a file being opened.
func (w *GoplsWrapper) DidOpen(uri string, content string, languageID string) error {
	params := &DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        DocumentURI(uri),
			LanguageID: languageID,
			Version:    1,
			Text:       content,
		},
	}
	return w.Notify("textDocument/didOpen", params)
}

// DidChange notifies the server about changes to a file.
func (w *GoplsWrapper) DidChange(uri string, version int, changes []TextDocumentContentChangeEvent) error {
	params := &DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Version: version,
		},
		ContentChanges: changes,
	}
	return w.Notify("textDocument/didChange", params)
}

// DidClose notifies the server about a file being closed.
func (w *GoplsWrapper) DidClose(uri string) error {
	params := &DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: DocumentURI(uri),
		},
	}
	return w.Notify("textDocument/didClose", params)
}

// DidSave notifies the server about a file being saved.
func (w *GoplsWrapper) DidSave(uri string, text *string) error {
	params := &DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: DocumentURI(uri),
		},
		Text: text,
	}
	return w.Notify("textDocument/didSave", params)
}

// Completion requests code completion items at the given position.
func (w *GoplsWrapper) Completion(uri string, line, character int) (*CompletionList, error) {
	params := &CompletionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Position: Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}
	var result CompletionList
	err := w.Call("textDocument/completion", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Definition requests the definition location of a symbol at the given position.
func (w *GoplsWrapper) Definition(uri string, line, character int) ([]Location, error) {
	params := &DefinitionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Position: Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}
	var result []Location
	err := w.Call("textDocument/definition", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Hover requests hover information at the given position.
func (w *GoplsWrapper) Hover(uri string, line, character int) (*Hover, error) {
	params := &HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Position: Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}
	var result Hover
	err := w.Call("textDocument/hover", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SignatureHelp requests signature help at the given position.
func (w *GoplsWrapper) SignatureHelp(uri string, line, character int) (*SignatureHelp, error) {
	params := &SignatureHelpParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Position: Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}
	var result SignatureHelp
	err := w.Call("textDocument/signatureHelp", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// References requests references to a symbol at the given position.
func (w *GoplsWrapper) References(uri string, line, character int, includeDeclaration bool) ([]Location, error) {
	params := &ReferenceParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Position: Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
		Context: ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}
	var result []Location
	err := w.Call("textDocument/references", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DocumentSymbol requests all symbols in a document.
func (w *GoplsWrapper) DocumentSymbol(uri string) ([]DocumentSymbol, error) {
	params := &DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{
			URI: DocumentURI(uri),
		},
	}
	var result []DocumentSymbol
	err := w.Call("textDocument/documentSymbol", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Format formats a document.
func (w *GoplsWrapper) Format(uri string) ([]TextEdit, error) {
	params := &DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{
			URI: DocumentURI(uri),
		},
		Options: FormattingOptions{
			TabSize:      4,
			InsertSpaces: true,
		},
	}
	var result []TextEdit
	err := w.Call("textDocument/formatting", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CodeAction requests code actions at the given range.
func (w *GoplsWrapper) CodeAction(uri string, startLine, startChar, endLine, endChar int, diagnostics []Diagnostic, only []CodeActionKind) ([]CodeAction, error) {
	params := &CodeActionParams{
		TextDocument: TextDocumentIdentifier{
			URI: DocumentURI(uri),
		},
		Range: Range{
			Start: Position{
				Line:      uint32(startLine),
				Character: uint32(startChar),
			},
			End: Position{
				Line:      uint32(endLine),
				Character: uint32(endChar),
			},
		},
		Context: CodeActionContext{
			Diagnostics: diagnostics,
			Only:        only,
		},
	}
	var result []CodeAction
	err := w.Call("textDocument/codeAction", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Rename renames a symbol at the given position.
func (w *GoplsWrapper) Rename(uri string, line, character int, newName string) (*WorkspaceEdit, error) {
	params := &RenameParams{
		TextDocument: TextDocumentIdentifier{
			URI: DocumentURI(uri),
		},
		Position: Position{
			Line:      uint32(line),
			Character: uint32(character),
		},
		NewName: newName,
	}
	var result WorkspaceEdit
	err := w.Call("textDocument/rename", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Implementation requests implementations of a symbol at the given position.
func (w *GoplsWrapper) Implementation(uri string, line, character int) ([]Location, error) {
	params := &ImplementationParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Position: Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}
	var result []Location
	err := w.Call("textDocument/implementation", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// TypeDefinition requests the type definition of a symbol at the given position.
func (w *GoplsWrapper) TypeDefinition(uri string, line, character int) ([]Location, error) {
	params := &TypeDefinitionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: DocumentURI(uri),
			},
			Position: Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}
	var result []Location
	err := w.Call("textDocument/typeDefinition", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// WorkspaceSymbol searches for symbols in the workspace.
func (w *GoplsWrapper) WorkspaceSymbol(query string) ([]SymbolInformation, error) {
	params := &WorkspaceSymbolParams{
		Query: query,
	}
	var result []SymbolInformation
	err := w.Call("workspace/symbol", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ExecuteCommand executes a command.
func (w *GoplsWrapper) ExecuteCommand(command string, args []interface{}) (interface{}, error) {
	params := &ExecuteCommandParams{
		Command:   command,
		Arguments: args,
	}
	var result interface{}
	err := w.Call("workspace/executeCommand", params, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Call sends a request to the gopls server and waits for a response.
func (w *GoplsWrapper) Call(method string, params interface{}, result interface{}) error {
	w.mu.Lock()
	id := w.nextID
	idStr := strconv.FormatInt(id, 10)
	w.nextID++
	ch := make(chan *Message, 1)
	w.responseCh[idStr] = ch
	w.mu.Unlock()

	// Create the request
	reqMsg := Message{
		Version: "2.0",
		ID:      []byte(idStr),
		Method:  method,
	}

	// Marshal params if not nil
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			w.mu.Lock()
			delete(w.responseCh, idStr)
			w.mu.Unlock()
			return fmt.Errorf("failed to marshal params: %v", err)
		}
		reqMsg.Params = data
	}

	// Marshal the request to JSON
	data, err := json.Marshal(reqMsg)
	if err != nil {
		w.mu.Lock()
		delete(w.responseCh, idStr)
		w.mu.Unlock()
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send the request
	_, err = w.stdin.Write(data)
	if err != nil {
		w.mu.Lock()
		delete(w.responseCh, idStr)
		w.mu.Unlock()
		return fmt.Errorf("failed to send request: %v", err)
	}

	// Wait for the response
	select {
	case resp := <-ch:
		w.mu.Lock()
		delete(w.responseCh, idStr)
		w.mu.Unlock()

		if resp.Error != nil {
			return fmt.Errorf("gopls error: %v", resp.Error.Message)
		}

		if result != nil && resp.Result != nil {
			err = json.Unmarshal(resp.Result, result)
			if err != nil {
				return fmt.Errorf("failed to unmarshal response: %v", err)
			}
		}

		return nil
	case <-w.ctx.Done():
		w.mu.Lock()
		delete(w.responseCh, idStr)
		w.mu.Unlock()
		return fmt.Errorf("context cancelled")
	}
}

// sendResponse sends a response to a request from gopls
func (w *GoplsWrapper) sendResponse(id string, result interface{}) error {
	resp := Message{
		Version: "2.0",
		ID:      []byte(id),
	}

	// Marshal result if not nil
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %v", err)
		}
		resp.Result = data
	}

	// Marshal the response to JSON
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %v", err)
	}

	// Send the response
	_, err = w.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send response: %v", err)
	}

	return nil
}

// sendErrorResponse sends an error response to a request from gopls
func (w *GoplsWrapper) sendErrorResponse(id string, err error) error {
	resp := Message{
		Version: "2.0",
		ID:      []byte(id),
		Error:   &Error{Code: -32603, Message: err.Error()},
	}

	// Marshal the response to JSON
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal error response: %v", err)
	}

	// Send the response
	_, err = w.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send error response: %v", err)
	}

	return nil
}

// Notify sends a notification to the gopls server.
func (w *GoplsWrapper) Notify(method string, params interface{}) error {
	// Create the notification
	notifMsg := Message{
		Version: "2.0",
		Method:  method,
	}

	// Marshal params if not nil
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %v", err)
		}
		notifMsg.Params = data
	}

	// Marshal the notification to JSON
	data, err := json.Marshal(notifMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %v", err)
	}

	// Send the notification
	_, err = w.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send notification: %v", err)
	}

	return nil
}

// RegisterHandler registers a handler for incoming requests and notifications.
func (w *GoplsWrapper) RegisterHandler(method string, handler func(interface{}) (interface{}, error)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[method] = handler
}

// Config wraps the configuration options for gopls.
type Config struct {
	EnvVars            map[string]string
	GoplsPath          string
	EnableGlobalConfig bool
	HoverKind          string
	BuildFlags         []string
	Env                []string
	DirectoryFilters   []string
	CompletionBudget   string
	Verbose            bool
}

// NewConfig creates a new default configuration.
func NewConfig() *Config {
	return &Config{
		EnvVars:            make(map[string]string),
		GoplsPath:          "gopls",
		EnableGlobalConfig: true,
		HoverKind:          "FullDocumentation",
		DirectoryFilters:   []string{"-node_modules"},
		CompletionBudget:   "100ms",
		Verbose:            false,
	}
}
