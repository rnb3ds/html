package html

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestPanicRecovery_NilProcessor tests that nil/closed processor is handled gracefully
func TestPanicRecovery_NilProcessor(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test nil processor
	var nilProc *Processor
	_, err = nilProc.Extract([]byte("<html>test</html>"))
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("nil processor: expected ErrProcessorClosed, got %v", err)
	}

	// Test closed processor
	processor.Close()
	_, err = processor.Extract([]byte("<html>test</html>"))
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("closed processor: expected ErrProcessorClosed, got %v", err)
	}
}

// TestPanicRecovery_LinksMethods tests that link extraction methods handle nil/closed processor
func TestPanicRecovery_LinksMethods(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test nil processor
	var nilProc *Processor
	_, err = nilProc.ExtractAllLinks([]byte("<html>test</html>"))
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("nil processor ExtractAllLinks: expected ErrProcessorClosed, got %v", err)
	}

	// Test closed processor
	processor.Close()
	_, err = processor.ExtractAllLinks([]byte("<html>test</html>"))
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("closed processor ExtractAllLinks: expected ErrProcessorClosed, got %v", err)
	}
}

// TestPanicRecovery_BatchMethods tests that batch methods handle nil/closed processor
func TestPanicRecovery_BatchMethods(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test nil processor
	var nilProc *Processor
	_, err = nilProc.ExtractBatch([][]byte{[]byte("<html>test</html>")})
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("nil processor ExtractBatch: expected ErrProcessorClosed, got %v", err)
	}

	// Test closed processor
	processor.Close()
	_, err = processor.ExtractBatch([][]byte{[]byte("<html>test</html>")})
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("closed processor ExtractBatch: expected ErrProcessorClosed, got %v", err)
	}
}

// TestPanicRecovery_ExtractFromFile tests that ExtractFromFile handles nil/closed processor
func TestPanicRecovery_ExtractFromFile(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Create temp file
	tmpFile, err := createTempHTMLFile()
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer cleanupTempFile(tmpFile)

	// Test nil processor
	var nilProc *Processor
	_, err = nilProc.ExtractFromFile(tmpFile)
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("nil processor ExtractFromFile: expected ErrProcessorClosed, got %v", err)
	}

	// Test closed processor
	processor.Close()
	_, err = processor.ExtractFromFile(tmpFile)
	if !errors.Is(err, ErrProcessorClosed) {
		t.Errorf("closed processor ExtractFromFile: expected ErrProcessorClosed, got %v", err)
	}
}

// TestPanicRecovery_ConcurrentSafety tests that concurrent processing is safe
func TestPanicRecovery_ConcurrentSafety(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	// Run multiple extractions concurrently to verify no race conditions
	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := processor.Extract([]byte("<html><body>test " + strconv.Itoa(idx) + "</body></html>"))
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("concurrent extraction failed: %v", err)
		}
	default:
	}
}

// TestPanicRecovery_EmptyInput tests that empty input is handled gracefully
func TestPanicRecovery_EmptyInput(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	// Empty input should not panic
	result, err := processor.Extract([]byte{})
	if err != nil {
		t.Errorf("empty input should not return error, got: %v", err)
	}
	if result == nil {
		t.Error("empty input should return non-nil result")
	}

	// Nil input should not panic
	result, err = processor.Extract(nil)
	if err != nil {
		t.Errorf("nil input should not return error, got: %v", err)
	}
	if result == nil {
		t.Error("nil input should return non-nil result")
	}
}

// TestPanicRecovery_LargeInput tests that large input is handled gracefully
func TestPanicRecovery_LargeInput(t *testing.T) {
	config := DefaultConfig()
	config.MaxInputSize = 1024 // 1KB limit
	processor, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	// Input exceeding limit should return error, not panic
	largeInput := make([]byte, 2048) // 2KB, exceeds 1KB limit
	for i := range largeInput {
		largeInput[i] = byte('a')
	}
	_, err = processor.Extract(largeInput)
	if !errors.Is(err, ErrInputTooLarge) {
		t.Errorf("large input: expected ErrInputTooLarge, got %v", err)
	}
}

// TestPanicRecovery_DeepNesting tests that deeply nested HTML is handled gracefully
func TestPanicRecovery_DeepNesting(t *testing.T) {
	config := DefaultConfig()
	config.MaxDepth = 50
	processor, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	// Create deeply nested HTML that exceeds limit
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 60; i++ {
		sb.WriteString("<div>")
	}
	sb.WriteString("content")
	for i := 0; i < 60; i++ {
		sb.WriteString("</div>")
	}
	sb.WriteString("</body></html>")

	_, err = processor.Extract([]byte(sb.String()))
	if !errors.Is(err, ErrMaxDepthExceeded) {
		t.Errorf("deep nesting: expected ErrMaxDepthExceeded, got %v", err)
	}
}

// TestPanicRecovery_InvalidHTML tests that invalid HTML is handled gracefully
func TestPanicRecovery_InvalidHTML(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	// Invalid HTML should not panic - it parser handles it gracefully
	result, err := processor.Extract([]byte("<<<>>>"))
	if err != nil {
		// Parser may return error or may try to parse anyway
		t.Logf("invalid HTML returned error: %v (acceptable)", err)
	}
	if result != nil {
		t.Logf("invalid HTML returned result: %+v", result.Text)
	}
}

func createTempHTMLFile() (string, error) {
	file, err := os.CreateTemp("", "test*.html")
	if err != nil {
		return "", err
	}
	content := []byte("<html><body>test content</body></html>")
	if err := os.WriteFile(file.Name(), content, 0644); err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func cleanupTempFile(filename string) {
	os.Remove(filename)
}
