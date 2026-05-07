package html_test

// batch_test.go - Tests for batch extraction functions
// Tests for Processor.ExtractBatch, ExtractBatchWithContext, ExtractBatchFiles, ExtractBatchFilesWithContext

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/cybergodev/html"
	"github.com/cybergodev/html/internal/testutil"
)

// TestExtractBatch tests the ExtractBatch processor method.
func TestExtractBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		docs        [][]byte
		wantLen     int
		wantErr     bool
		checkClosed bool
	}{
		{
			name:    "basic batch extraction",
			docs:    [][]byte{[]byte(`<html><body><p>Document 1</p></body></html>`), []byte(`<html><body><p>Document 2</p></body></html>`), []byte(`<html><body><p>Document 3</p></body></html>`)},
			wantLen: 3,
		},
		{
			name:    "empty batch nil",
			docs:    nil,
			wantLen: 0,
		},
		{
			name:    "empty batch slice",
			docs:    [][]byte{},
			wantLen: 0,
		},
		{
			name:    "batch with invalid HTML",
			docs:    [][]byte{[]byte(`<html><body><p>Valid</p></body></html>`), []byte(""), []byte(`<html><body><p>Also valid</p></body></html>`)},
			wantLen: 3,
		},
		{
			name:        "closed processor",
			docs:        [][]byte{[]byte(`<html><body><p>Content</p></body></html>`)},
			wantErr:     true,
			checkClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := html.New()
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			if tt.checkClosed {
				p.Close()
			} else {
				defer p.Close()
			}

			br := p.ExtractBatch(tt.docs)
			if tt.wantErr {
				if br.Failed == 0 {
					t.Error("Expected error")
				}
				if !errors.Is(br.Errors[0], html.ErrProcessorClosed) {
					t.Errorf("Expected ErrProcessorClosed, got: %v", br.Errors[0])
				}
				return
			}

			if br.Failed > 0 {
				t.Fatalf("ExtractBatch() failed: %v", br.Errors[0])
			}

			if len(br.Results) != tt.wantLen {
				t.Errorf("Expected %d results, got %d", tt.wantLen, len(br.Results))
			}
		})
	}
}

// TestExtractBatchWithConfig tests ExtractBatch with custom configuration.
func TestExtractBatchWithConfig(t *testing.T) {
	t.Parallel()

	cfg := html.DefaultConfig()
	cfg.PreserveImages = false
	p := testutil.NewTestProcessor(t, cfg)

	docs := [][]byte{[]byte(`<html><body><p>Content</p><img src="test.jpg"/></body></html>`)}

	br := p.ExtractBatch(docs)
	if br.Failed > 0 {
		t.Fatalf("ExtractBatch() with config failed: %v", br.Errors[0])
	}

	if len(br.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(br.Results))
	}
	if len(br.Results[0].Images) != 0 {
		t.Errorf("Expected no images with PreserveImages=false")
	}
}

// TestExtractBatchWithContext tests the ExtractBatchWithContext processor method.
func TestExtractBatchWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		docs         [][]byte
		ctx          context.Context
		cancelBefore bool
		wantSuccess  int
		wantFailed   int
	}{
		{
			name:        "successful batch with context",
			docs:        [][]byte{[]byte(`<html><body><p>Doc 1</p></body></html>`), []byte(`<html><body><p>Doc 2</p></body></html>`)},
			ctx:         context.Background(),
			wantSuccess: 2,
		},
		{
			name:         "cancelled context",
			docs:         createNDocs(100),
			cancelBefore: true,
		},
		{
			name:        "empty batch with context",
			docs:        nil,
			ctx:         context.Background(),
			wantSuccess: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testutil.NewTestProcessor(t)

			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			if tt.cancelBefore {
				cctx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cctx
			}

			result := p.ExtractBatchWithContext(ctx, tt.docs)

			if !tt.cancelBefore {
				if result.Success != tt.wantSuccess {
					t.Errorf("Expected %d successful, got %d", tt.wantSuccess, result.Success)
				}
				if result.Failed != tt.wantFailed {
					t.Errorf("Expected %d failed, got %d", tt.wantFailed, result.Failed)
				}
			}

			if tt.docs == nil {
				if len(result.Results) != 0 || len(result.Errors) != 0 {
					t.Errorf("Expected empty results for nil input")
				}
			}
		})
	}
}

// TestExtractBatchWithContextTimeout tests context timeout behavior.
func TestExtractBatchWithContextTimeout(t *testing.T) {
	t.Parallel()

	p := testutil.NewTestProcessor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Wait for context to expire

	docs := createNDocs(50)
	result := p.ExtractBatchWithContext(ctx, docs)

	// With expired context, operations should be cancelled
	if result.Cancelled == 0 && result.Success == 0 {
		t.Logf("Cancelled: %d, Success: %d, Failed: %d", result.Cancelled, result.Success, result.Failed)
	}
}

// TestExtractBatchFiles tests the ExtractBatchFiles processor method.
func TestExtractBatchFiles(t *testing.T) {
	t.Parallel()

	t.Run("batch file extraction", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := createTempHTMLFiles(t, tmpDir, 3)

		p := testutil.NewTestProcessor(t)
		br := p.ExtractBatchFiles(files)
		if br.Failed > 0 {
			t.Fatalf("ExtractBatchFiles() failed: %v", br.Errors[0])
		}

		if len(br.Results) != len(files) {
			t.Errorf("Expected %d results, got %d", len(files), len(br.Results))
		}
	})

	t.Run("batch with non-existent file", func(t *testing.T) {
		p := testutil.NewTestProcessor(t)
		br := p.ExtractBatchFiles([]string{"non-existent-file.html"})
		if br.Failed == 0 {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("empty file list", func(t *testing.T) {
		p := testutil.NewTestProcessor(t)
		br := p.ExtractBatchFiles(nil)
		if br.Failed > 0 {
			t.Fatalf("ExtractBatchFiles(nil) failed: %v", br.Errors[0])
		}
		if len(br.Results) != 0 {
			t.Errorf("Expected 0 results for nil input, got %d", len(br.Results))
		}
	})

	t.Run("closed processor", func(t *testing.T) {
		p, _ := html.New()
		p.Close()

		br := p.ExtractBatchFiles([]string{"test.html"})
		if br.Failed == 0 {
			t.Error("Expected error on closed processor")
		}
	})
}

// TestExtractBatchFilesWithContext tests the ExtractBatchFilesWithContext processor method.
func TestExtractBatchFilesWithContext(t *testing.T) {
	t.Parallel()

	t.Run("batch file extraction with context", func(t *testing.T) {
		p := testutil.NewTestProcessor(t)
		tmpDir := t.TempDir()
		files := createTempHTMLFiles(t, tmpDir, 2)

		result := p.ExtractBatchFilesWithContext(context.Background(), files)

		if len(result.Results) != len(files) {
			t.Errorf("Expected %d results, got %d", len(files), len(result.Results))
		}
		if result.Success != len(files) {
			t.Errorf("Expected %d successful, got %d", len(files), result.Success)
		}
	})

	t.Run("batch files with errors", func(t *testing.T) {
		p := testutil.NewTestProcessor(t)
		tmpDir := t.TempDir()

		validFile := tmpDir + "/valid.html"
		os.WriteFile(validFile, []byte(`<html><body><p>Valid</p></body></html>`), 0644)

		files := []string{validFile, "non-existent-file.html"}

		result := p.ExtractBatchFilesWithContext(context.Background(), files)

		if result.Success == 0 {
			t.Error("Expected at least one successful extraction")
		}
		if result.Failed == 0 {
			t.Error("Expected at least one failed extraction")
		}
	})
}

// TestBatchResultStructure tests the BatchResult structure fields.
func TestBatchResultStructure(t *testing.T) {
	t.Parallel()

	p := testutil.NewTestProcessor(t)
	docs := [][]byte{
		[]byte(`<html><body><p>Doc 1</p></body></html>`),
		[]byte(`<html><body><p>Doc 2</p></body></html>`),
	}

	result := p.ExtractBatchWithContext(context.Background(), docs)

	if result.Results == nil {
		t.Error("Results slice should not be nil")
	}
	if result.Errors == nil {
		t.Error("Errors slice should not be nil")
	}

	total := result.Success + result.Failed + result.Cancelled
	if total != len(docs) {
		t.Errorf("Success + Failed + Cancelled = %d, expected %d", total, len(docs))
	}
}

// TestConcurrentBatchOperations tests concurrent batch operations.
func TestConcurrentBatchOperations(t *testing.T) {
	t.Parallel()

	p := testutil.NewTestProcessor(t)
	docs := [][]byte{[]byte(`<html><body><p>Content</p></body></html>`)}

	errs := testutil.RunConcurrent(10, func(int) error {
		br := p.ExtractBatch(docs)
		if br.Failed > 0 {
			return br.Errors[0]
		}
		return nil
	})

	for i, err := range errs {
		if err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}
}

// TestBatchWithLargeInput tests batch processing with large inputs.
func TestBatchWithLargeInput(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping large input test in short mode")
	}

	p := testutil.NewTestProcessor(t)
	docs := createNDocs(100)

	br := p.ExtractBatch(docs)
	if br.Failed > 0 {
		t.Fatalf("ExtractBatch() with large input failed: %v", br.Errors[0])
	}

	if len(br.Results) != 100 {
		t.Errorf("Expected 100 results, got %d", len(br.Results))
	}

	successCount := 0
	for _, result := range br.Results {
		if result != nil {
			successCount++
		}
	}

	if successCount != 100 {
		t.Errorf("Expected 100 successful extractions, got %d", successCount)
	}
}

// Helper functions

func createNDocs(n int) [][]byte {
	docs := make([][]byte, n)
	content := []byte(`<html><body><p>Document content for testing batch processing.</p></body></html>`)
	for i := range docs {
		docs[i] = content
	}
	return docs
}

func createTempHTMLFiles(t *testing.T, tmpDir string, count int) []string {
	t.Helper()
	files := make([]string, count)
	for i := range count {
		files[i] = tmpDir + "/file" + string(rune('A'+i)) + ".html"
		content := []byte(`<html><body><h1>File ` + string(rune('A'+i)) + `</h1></body></html>`)
		if err := os.WriteFile(files[i], content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	return files
}
