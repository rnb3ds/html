package html_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cybergodev/html"
)

func TestExtractBatch(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("empty batch", func(t *testing.T) {
		results, err := p.ExtractBatch([]string{}, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("ExtractBatch() returned %d results, want 0", len(results))
		}
	})

	t.Run("single item", func(t *testing.T) {
		htmlContents := []string{
			`<html><body><p>Test 1</p></body></html>`,
		}

		results, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("ExtractBatch() returned %d results, want 1", len(results))
		}

		if !strings.Contains(results[0].Text, "Test 1") {
			t.Errorf("Result text = %q, want to contain %q", results[0].Text, "Test 1")
		}
	})

	t.Run("multiple items", func(t *testing.T) {
		htmlContents := []string{
			`<html><body><p>Test 1</p></body></html>`,
			`<html><body><p>Test 2</p></body></html>`,
			`<html><body><p>Test 3</p></body></html>`,
		}

		results, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatch() failed: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("ExtractBatch() returned %d results, want 3", len(results))
		}

		for i, result := range results {
			expected := "Test " + string(rune('1'+i))
			if !strings.Contains(result.Text, expected) {
				t.Errorf("Result[%d] text = %q, want to contain %q", i, result.Text, expected)
			}
		}
	})

	t.Run("partial failure", func(t *testing.T) {
		htmlContents := []string{
			`<html><body><p>Valid</p></body></html>`,
			strings.Repeat("x", 100*1024*1024), // Too large
			`<html><body><p>Also valid</p></body></html>`,
		}

		results, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractBatch() should return error for partial failure")
		}

		if len(results) != 3 {
			t.Fatalf("ExtractBatch() returned %d results, want 3", len(results))
		}

		if results[0] == nil {
			t.Error("Result[0] should not be nil")
		}
		if results[1] != nil {
			t.Error("Result[1] should be nil (failed)")
		}
		if results[2] == nil {
			t.Error("Result[2] should not be nil")
		}
	})

	t.Run("all failures", func(t *testing.T) {
		htmlContents := []string{
			strings.Repeat("x", 100*1024*1024),
			strings.Repeat("y", 100*1024*1024),
		}

		_, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractBatch() should return error when all items fail")
		}
	})

	t.Run("after close", func(t *testing.T) {
		p2 := html.NewWithDefaults()
		p2.Close()

		_, err := p2.ExtractBatch([]string{"<html></html>"}, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractBatch() should fail after Close()")
		}
	})
}

func TestExtractBatchFiles(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	t.Run("empty batch", func(t *testing.T) {
		results, err := p.ExtractBatchFiles([]string{}, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatchFiles() failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("ExtractBatchFiles() returned %d results, want 0", len(results))
		}
	})

	t.Run("non-existent files", func(t *testing.T) {
		filePaths := []string{
			"nonexistent1.html",
			"nonexistent2.html",
		}

		_, err := p.ExtractBatchFiles(filePaths, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractBatchFiles() should fail with non-existent files")
		}
	})

	t.Run("valid files", func(t *testing.T) {
		// Create temporary test files
		tmpDir := t.TempDir()

		file1 := filepath.Join(tmpDir, "test1.html")
		file2 := filepath.Join(tmpDir, "test2.html")

		if err := os.WriteFile(file1, []byte(`<html><body><p>File 1</p></body></html>`), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := os.WriteFile(file2, []byte(`<html><body><p>File 2</p></body></html>`), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		filePaths := []string{file1, file2}

		results, err := p.ExtractBatchFiles(filePaths, html.DefaultExtractConfig())
		if err != nil {
			t.Fatalf("ExtractBatchFiles() failed: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("ExtractBatchFiles() returned %d results, want 2", len(results))
		}

		if !strings.Contains(results[0].Text, "File 1") {
			t.Errorf("Result[0] text = %q, want to contain %q", results[0].Text, "File 1")
		}
		if !strings.Contains(results[1].Text, "File 2") {
			t.Errorf("Result[1] text = %q, want to contain %q", results[1].Text, "File 2")
		}
	})

	t.Run("partial failure with files", func(t *testing.T) {
		tmpDir := t.TempDir()

		validFile := filepath.Join(tmpDir, "valid.html")
		if err := os.WriteFile(validFile, []byte(`<html><body><p>Valid</p></body></html>`), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		filePaths := []string{
			validFile,
			"nonexistent.html",
		}

		results, err := p.ExtractBatchFiles(filePaths, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractBatchFiles() should return error for partial failure")
		}

		if len(results) != 2 {
			t.Fatalf("ExtractBatchFiles() returned %d results, want 2", len(results))
		}

		if results[0] == nil {
			t.Error("Result[0] should not be nil")
		}
		if results[1] != nil {
			t.Error("Result[1] should be nil (failed)")
		}
	})

	t.Run("after close", func(t *testing.T) {
		p2 := html.NewWithDefaults()
		p2.Close()

		_, err := p2.ExtractBatchFiles([]string{"test.html"}, html.DefaultExtractConfig())
		if err == nil {
			t.Fatal("ExtractBatchFiles() should fail after Close()")
		}
	})
}

func TestBatchConcurrency(t *testing.T) {
	t.Parallel()

	p := html.NewWithDefaults()
	defer p.Close()

	// Create a batch large enough to test worker pool
	htmlContents := make([]string, 20)
	for i := range htmlContents {
		htmlContents[i] = `<html><body><p>Test content</p></body></html>`
	}

	results, err := p.ExtractBatch(htmlContents, html.DefaultExtractConfig())
	if err != nil {
		t.Fatalf("ExtractBatch() failed: %v", err)
	}

	if len(results) != 20 {
		t.Fatalf("ExtractBatch() returned %d results, want 20", len(results))
	}

	for i, result := range results {
		if result == nil {
			t.Errorf("Result[%d] is nil", i)
		}
	}
}
