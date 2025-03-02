package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/flaticols/mcpgopls/internal/gopls"
	"github.com/flaticols/mcpgopls/internal/mcp"
	"github.com/jessevdk/go-flags"
	"github.com/lmittmann/tint"
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
	
	// Configure colored logger
	logLevel := slog.LevelInfo
	if opts.Verbose {
		logLevel = slog.LevelDebug
	}
	
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.Kitchen,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Ensure root directory is absolute
	rootDir, err := filepath.Abs(opts.RootDir)
	if err != nil {
		slog.Error("Failed to resolve root directory", "error", err)
		os.Exit(1)
	}

	// Configure gopls
	config := gopls.NewConfig()
	if opts.Verbose {
		config.Verbose = true
	}

	// Create the server
	server, err := mcp.NewClaudeGoplsServer(config, rootDir)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Start gopls
	err = server.Start()
	if err != nil {
		slog.Error("Failed to start gopls", "error", err)
		os.Exit(1)
	}
	defer server.Stop()

	// Start command server
	slog.Info("Starting MCP-GOPLS server", 
		"host", opts.ServerListen,
		"port", opts.ServerPort,
		"rootDir", rootDir)

	// Create and configure HTTP server to handle commands
	if err := setupCommandServer(opts.ServerListen, opts.ServerPort, server); err != nil {
		slog.Error("Failed to start command server", "error", err)
		os.Exit(1)
	}
}

// setupCommandServer creates and starts an HTTP server that handles command execution
func setupCommandServer(host string, port int, server *mcp.ClaudeGoplsServer) error {
	return nil
}
