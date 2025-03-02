package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/flaticols/mcpgopls/internal/gopls"
	"github.com/flaticols/mcpgopls/internal/mcp"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	RootDir      string `short:"r" long:"root" description:"Root directory for the workspace" required:"true"`
	Verbose      bool   `short:"v" long:"verbose" description:"Enable verbose logging"`
	ServerPort   int    `short:"p" long:"port" description:"Server port" default:"8123"`
	ServerListen string `short:"l" long:"listen" description:"Server listen address" default:"127.0.0.1"`
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		// Handle flag errors specifically
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// Ensure root directory is absolute
	rootDir, err := filepath.Abs(opts.RootDir)
	if err != nil {
		log.Fatalf("Failed to resolve root directory: %v", err)
	}

	// Configure gopls
	config := gopls.NewConfig()
	if opts.Verbose {
		config.Verbose = true
	}

	// Create the server
	server, err := mcp.NewClaudeGoplsServer(config, rootDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start gopls
	err = server.Start()
	if err != nil {
		log.Fatalf("Failed to start gopls: %v", err)
	}
	defer server.Stop()

	// Start command server
	fmt.Printf("Starting MCP-GOPLS server on %s:%d with root directory: %s\n",
		opts.ServerListen, opts.ServerPort, rootDir)

	// Create and configure HTTP server to handle commands
	if err := setupCommandServer(opts.ServerListen, opts.ServerPort, server); err != nil {
		log.Fatalf("Failed to start command server: %v", err)
	}
}

// setupCommandServer creates and starts an HTTP server that handles command execution
func setupCommandServer(host string, port int, server *mcp.ClaudeGoplsServer) error {
	return nil
}
