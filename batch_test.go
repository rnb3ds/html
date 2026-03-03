package html_test

// batch_test.go - Tests for batch extraction functions
// Tests for Processor.ExtractBatch, ExtractBatchWithContext, ExtractBatchFiles, ExtractBatchFilesWithContext

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// TestExtractBatch tests the ExtractBatch processor method.
func TestExtractBatch(t *testing.T) {
	t.Parallel()

	t.Run("basic batch extraction", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		docs := [][]byte{
			[]byte(`<html><body><p>Document 1</p></body></html>`),
			[]byte(`<html><body><p>Document 2</p></body></html>`),
			[]byte(`<html><body><p>Document 3</p></body></html>`),
		}

		results, err := p.ExtractBatch(docs)
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}

		if len(results) != len(docs) {
			t.Errorf("Expected %d results, got %d", len(docs), len(results))
		}

		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d is nil", i)
				continue
			}
			expected := "Document"
			if !containsString(result.Text, expected) {
				t.Errorf("Result %d: expected text to contain %q, got %q", i, expected, result.Text)
			}
		}
	})

	t.Run("empty batch", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		results, err := p.ExtractBatch(nil)
		if err != nil {
			t.Fatalf("ExtractBatch(nil) failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for nil input, got %d", len(results))
		}

		results, err = p.ExtractBatch([][]byte{})
		if err != nil {
			t.Fatalf("ExtractBatch([]) failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty slice, got %d", len(results))
		}
	})

	t.Run("batch with invalid HTML", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		docs := [][]byte{
			[]byte(`<html><body><p>Valid</p></body></html>`),
			[]byte(""), // Empty
			[]byte(`<html><body><p>Also valid</p></body></html>`),
		}

		results, err := p.ExtractBatch(docs)
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}

		if len(results) != len(docs) {
			t.Errorf("Expected %d results, got %d", len(docs), len(results))
		}
	})
}

// TestExtractBatchWithContext tests the ExtractBatchWithContext processor method.
func TestExtractBatchWithContext(t *testing.T) {
	t.Parallel()

	t.Run("successful batch with context", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ctx := context.Background()
		docs := [][]byte{
			[]byte(`<html><body><p>Doc 1</p></body></html>`),
			[]byte(`<html><body><p>Doc 2</p></body></html>`),
		}

		result := p.ExtractBatchWithContext(ctx, docs)

		if len(result.Results) != len(docs) {
			t.Errorf("Expected %d results, got %d", len(docs), len(result.Results))
		}
		if result.Success != len(docs) {
			t.Errorf("Expected %d successful, got %d", len(docs), result.Success)
		}
		if result.Failed != 0 {
			t.Errorf("Expected 0 failed, got %d", result.Failed)
		}
		if result.Cancelled != 0 {
			t.Errorf("Expected 0 cancelled, got %d", result.Cancelled)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		docs := make([][]byte, 100)
		for i := range docs {
			docs[i] = []byte(`<html><body><p>Content</p></body></html>`)
		}

		result := p.ExtractBatchWithContext(ctx, docs)

		// Some or all should be cancelled
		if result.Cancelled == 0 && result.Success == 0 {
			t.Error("Expected some operations to be cancelled or succeed")
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for context to expire
		time.Sleep(10 * time.Millisecond)

		docs := make([][]byte, 50)
		for i := range docs {
			docs[i] = []byte(`<html><body><p>Content</p></body></html>`)
		}

		result := p.ExtractBatchWithContext(ctx, docs)

		// With expired context, operations should be cancelled
		if result.Cancelled == 0 && result.Success == 0 {
			t.Logf("Cancelled: %d, Success: %d, Failed: %d",
				result.Cancelled, result.Success, result.Failed)
		}
	})

	t.Run("empty batch with context", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ctx := context.Background()
		result := p.ExtractBatchWithContext(ctx, nil)

		if len(result.Results) != 0 {
			t.Errorf("Expected 0 results for nil input, got %d", len(result.Results))
		}
		if len(result.Errors) != 0 {
			t.Errorf("Expected 0 errors for nil input, got %d", len(result.Errors))
		}
	})
}

// TestExtractBatchFiles tests the ExtractBatchFiles processor method.
func TestExtractBatchFiles(t *testing.T) {
	t.Parallel()

	t.Run("batch file extraction", func(t *testing.T) {
		// Create temp files
		tmpDir := t.TempDir()
		files := []string{
			tmpDir + "/file1.html",
			tmpDir + "/file2.html",
			tmpDir + "/file3.html",
		}

		for i, file := range files {
			content := []byte(`<html><body><h1>File ` + string(rune('A'+i)) + `</h1></body></html>`)
			if err := os.WriteFile(file, content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		results, err := p.ExtractBatchFiles(files)
		if err != nil {
			t.Fatalf("ExtractBatchFiles() failed: %v", err)
		}

		if len(results) != len(files) {
			t.Errorf("Expected %d results, got %d", len(files), len(results))
		}

		for i, result := range results {
			if result == nil {
				t.Errorf("Result %d is nil", i)
			}
		}
	})

	t.Run("batch with non-existent file", func(t *testing.T) {
		files := []string{
			"non-existent-file.html",
		}

		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		results, err := p.ExtractBatchFiles(files)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		_ = results // May be nil on error
	})

	t.Run("empty file list", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		results, err := p.ExtractBatchFiles(nil)
		if err != nil {
			t.Fatalf("ExtractBatchFiles(nil) failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for nil input, got %d", len(results))
		}
	})
}

// TestExtractBatchFilesWithContext tests the ExtractBatchFilesWithContext processor method.
func TestExtractBatchFilesWithContext(t *testing.T) {
	t.Parallel()

	t.Run("batch file extraction with context", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ctx := context.Background()

		// Create temp files
		tmpDir := t.TempDir()
		files := []string{
			tmpDir + "/file1.html",
			tmpDir + "/file2.html",
		}

		for i, file := range files {
			content := []byte(`<html><body><p>Content ` + string(rune('1'+i)) + `</p></body></html>`)
			if err := os.WriteFile(file, content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		result := p.ExtractBatchFilesWithContext(ctx, files)

		if len(result.Results) != len(files) {
			t.Errorf("Expected %d results, got %d", len(files), len(result.Results))
		}
		if result.Success != len(files) {
			t.Errorf("Expected %d successful, got %d", len(files), result.Success)
		}
	})

	t.Run("batch files with cancelled context", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		files := make([]string, 20)
		tmpDir := t.TempDir()
		for i := range files {
			files[i] = tmpDir + "/file.html"
		}

		result := p.ExtractBatchFilesWithContext(ctx, files)

		// With cancelled context, operations should be cancelled
		totalProcessed := result.Cancelled + result.Success + result.Failed
		if totalProcessed != len(files) {
			t.Logf("Cancelled: %d, Success: %d, Failed: %d",
				result.Cancelled, result.Success, result.Failed)
		}
	})

	t.Run("batch files with errors", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ctx := context.Background()

		// Mix of valid and invalid files
		tmpDir := t.TempDir()
		validFile := tmpDir + "/valid.html"
		if err := os.WriteFile(validFile, []byte(`<html><body><p>Valid</p></body></html>`), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		files := []string{
			validFile,
			"non-existent-file.html",
		}

		result := p.ExtractBatchFilesWithContext(ctx, files)

		if result.Success == 0 {
			t.Error("Expected at least one successful extraction")
		}
		if result.Failed == 0 {
			t.Error("Expected at least one failed extraction")
		}
	})
}

// TestProcessorExtractBatch tests the Processor.ExtractBatch method.
func TestProcessorExtractBatch(t *testing.T) {
	t.Parallel()

	t.Run("processor batch extraction", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		docs := [][]byte{
			[]byte(`<html><body><p>Doc A</p></body></html>`),
			[]byte(`<html><body><p>Doc B</p></body></html>`),
		}

		results, err := p.ExtractBatch(docs)
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}

		if len(results) != len(docs) {
			t.Errorf("Expected %d results, got %d", len(docs), len(results))
		}
	})

	t.Run("processor batch with custom config", func(t *testing.T) {
		cfg := html.DefaultConfig()
		cfg.PreserveImages = false
		p, err := html.New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		docs := [][]byte{
			[]byte(`<html><body><p>Content</p><img src="test.jpg"/></body></html>`),
		}

		results, err := p.ExtractBatch(docs)
		if err != nil {
			t.Fatalf("ExtractBatch() with config failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if len(results[0].Images) != 0 {
			t.Errorf("Expected no images with PreserveImages=false")
		}
	})

	t.Run("processor batch on closed processor", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		p.Close()

		docs := [][]byte{
			[]byte(`<html><body><p>Content</p></body></html>`),
		}

		_, err := p.ExtractBatch(docs)
		if err == nil {
			t.Error("Expected error on closed processor")
		}
		if !errors.Is(err, html.ErrProcessorClosed) {
			t.Errorf("Expected ErrProcessorClosed, got: %v", err)
		}
	})
}

// TestProcessorExtractBatchWithContext tests the Processor.ExtractBatchWithContext method.
func TestProcessorExtractBatchWithContext(t *testing.T) {
	t.Parallel()

	t.Run("processor batch with context", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		ctx := context.Background()
		docs := [][]byte{
			[]byte(`<html><body><p>Content</p></body></html>`),
		}

		result := p.ExtractBatchWithContext(ctx, docs)

		if result.Success != 1 {
			t.Errorf("Expected 1 successful, got %d", result.Success)
		}
	})

	t.Run("processor batch with context on closed processor", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		p.Close()

		ctx := context.Background()
		docs := [][]byte{
			[]byte(`<html><body><p>Content</p></body></html>`),
		}

		result := p.ExtractBatchWithContext(ctx, docs)

		if result.Failed != 1 {
			t.Errorf("Expected 1 failed, got %d", result.Failed)
		}
		if result.Errors[0] == nil {
			t.Error("Expected error in Errors slice")
		}
	})
}

// TestProcessorExtractBatchFiles tests the Processor.ExtractBatchFiles method.
func TestProcessorExtractBatchFiles(t *testing.T) {
	t.Parallel()

	t.Run("processor batch file extraction", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		tmpDir := t.TempDir()
		files := []string{
			tmpDir + "/file1.html",
			tmpDir + "/file2.html",
		}

		for i, file := range files {
			content := []byte(`<html><body><p>File ` + string(rune('1'+i)) + `</p></body></html>`)
			if err := os.WriteFile(file, content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		results, err := p.ExtractBatchFiles(files)
		if err != nil {
			t.Fatalf("ExtractBatchFiles() failed: %v", err)
		}

		if len(results) != len(files) {
			t.Errorf("Expected %d results, got %d", len(files), len(results))
		}
	})

	t.Run("processor batch files on closed processor", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		p.Close()

		files := []string{"test.html"}

		_, err := p.ExtractBatchFiles(files)
		if err == nil {
			t.Error("Expected error on closed processor")
		}
	})
}

// TestProcessorExtractBatchFilesWithContext tests the Processor.ExtractBatchFilesWithContext method.
func TestProcessorExtractBatchFilesWithContext(t *testing.T) {
	t.Parallel()

	t.Run("processor batch files with context", func(t *testing.T) {
		p, err := html.New(html.DefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		ctx := context.Background()
		tmpDir := t.TempDir()
		file := tmpDir + "/test.html"
		if err := os.WriteFile(file, []byte(`<html><body><p>Content</p></body></html>`), 0644); err != nil {
			t.Fatal(err)
		}

		result := p.ExtractBatchFilesWithContext(ctx, []string{file})

		if result.Success != 1 {
			t.Errorf("Expected 1 successful, got %d", result.Success)
		}
	})
}

// TestBatchResultStructure tests the BatchResult structure fields.
func TestBatchResultStructure(t *testing.T) {
	t.Parallel()

	t.Run("verify batch result fields", func(t *testing.T) {
		p, _ := html.New(html.DefaultConfig())
		defer p.Close()

		ctx := context.Background()
		docs := [][]byte{
			[]byte(`<html><body><p>Doc 1</p></body></html>`),
			[]byte(`<html><body><p>Doc 2</p></body></html>`),
		}

		result := p.ExtractBatchWithContext(ctx, docs)

		// Verify Results slice
		if result.Results == nil {
			t.Error("Results slice should not be nil")
		}

		// Verify Errors slice
		if result.Errors == nil {
			t.Error("Errors slice should not be nil")
		}

		// Verify counts
		total := result.Success + result.Failed + result.Cancelled
		if total != len(docs) {
			t.Errorf("Success + Failed + Cancelled = %d, expected %d", total, len(docs))
		}
	})
}

// TestConcurrentBatchOperations tests concurrent batch operations.
func TestConcurrentBatchOperations(t *testing.T) {
	t.Parallel()

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	const numGoroutines = 10
	var wg sync.WaitGroup

	docs := [][]byte{
		[]byte(`<html><body><p>Content</p></body></html>`),
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.ExtractBatch(docs)
			if err != nil {
				t.Errorf("Concurrent ExtractBatch failed: %v", err)
			}
		}()
	}

	wg.Wait()
}

// TestBatchWithLargeInput tests batch processing with large inputs.
func TestBatchWithLargeInput(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping large input test in short mode")
	}

	p, err := html.New(html.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create batch of 100 documents
	docs := make([][]byte, 100)
	for i := range docs {
		docs[i] = []byte(`<html><body><p>Document content for testing batch processing.</p></body></html>`)
	}

	results, err := p.ExtractBatch(docs)
	if err != nil {
		t.Fatalf("ExtractBatch() with large input failed: %v", err)
	}

	if len(results) != 100 {
		t.Errorf("Expected 100 results, got %d", len(results))
	}

	successCount := 0
	for _, result := range results {
		if result != nil {
			successCount++
		}
	}

	if successCount != 100 {
		t.Errorf("Expected 100 successful extractions, got %d", successCount)
	}
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
