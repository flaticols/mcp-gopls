package gopls

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoplsWrapper(t *testing.T) {
	// Skip test if gopls is not installed
	_, err := exec.LookPath("gopls")
	if err != nil {
		t.Skip("Skipping test as gopls is not installed")
	}

	wrapper, err := NewGoplsWrapper()
	require.NoError(t, err)
	require.NotNil(t, wrapper)
	assert.NotNil(t, wrapper.cmd)
	assert.NotNil(t, wrapper.stdin)
	assert.NotNil(t, wrapper.stdout)
	assert.NotNil(t, wrapper.stderr)
	assert.NotNil(t, wrapper.handlers)
	assert.NotNil(t, wrapper.responseCh)
	assert.NotNil(t, wrapper.ctx)
	assert.NotNil(t, wrapper.cancel)
}

func TestHeaderWriter(t *testing.T) {
	// Create a buffer to write to
	buffer := make([]byte, 0)
	mockWriter := mockWriter{
		writeFn: func(p []byte) (int, error) {
			buffer = append(buffer, p...)
			return len(p), nil
		},
	}

	// Create a HeaderWriter with the mock writer
	headerWriter := NewHeaderWriter(mockWriter)
	
	// Test writing data
	testData := []byte("test data")
	n, err := headerWriter.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	
	// Check the written content includes headers
	expectedHeader := "Content-Length: 9\r\n\r\n"
	expectedOutput := expectedHeader + "test data"
	assert.Equal(t, expectedOutput, string(buffer))
}

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "gopls", config.GoplsPath)
	assert.True(t, config.EnableGlobalConfig)
	assert.Equal(t, "FullDocumentation", config.HoverKind)
	assert.Contains(t, config.DirectoryFilters, "-node_modules")
	assert.Equal(t, "100ms", config.CompletionBudget)
}

// mockWriter is a mock implementation of io.Writer for testing
type mockWriter struct {
	writeFn func(p []byte) (n int, err error)
}

func (m mockWriter) Write(p []byte) (n int, err error) {
	return m.writeFn(p)
}

func TestGoplsInitialization(t *testing.T) {
	// Skip test if gopls is not installed
	_, err := exec.LookPath("gopls")
	if err != nil {
		t.Skip("Skipping test as gopls is not installed")
	}

	wrapper, err := NewGoplsWrapper()
	require.NoError(t, err)
	
	// Start the gopls process
	err = wrapper.Start()
	require.NoError(t, err)
	
	// Clean up after the test
	defer func() {
		err := wrapper.Stop()
		assert.NoError(t, err)
	}()
	
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gopls-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Initialize gopls with the temporary directory
	result, err := wrapper.Initialize(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Capabilities)
}