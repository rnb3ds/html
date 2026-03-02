package html

import (
	"context"
	"fmt"
	"sync"

	"github.com/cybergodev/html/internal"
)

// BatchResult holds the results of a batch extraction operation.
type BatchResult struct {
	// Results contains the extraction results for each input item.
	// Nil results indicate that the extraction failed or was cancelled.
	Results []*Result
	// Errors contains the error for each input item, if any.
	// The index corresponds to the index in the input slice.
	Errors []error
	// Success is the count of successful extractions.
	Success int
	// Failed is the count of failed extractions.
	Failed int
	// Cancelled is the count of items that were not processed due to context cancellation.
	Cancelled int
}

// ExtractBatch extracts content from multiple HTML byte slices concurrently.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// Returns a slice of results and a combined error if any extractions failed.
func (p *Processor) ExtractBatch(htmlContents [][]byte) ([]*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(htmlContents) == 0 {
		return []*Result{}, nil
	}

	config := p.getExtractConfig()

	results := make([]*Result, len(htmlContents))
	errs := make([]error, len(htmlContents))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, content := range htmlContents {
		// Acquire semaphore BEFORE creating goroutine to limit concurrent goroutines
		sem <- struct{}{}

		wg.Add(1)
		go func(idx int, htmlBytes []byte) {
			defer func() {
				<-sem
				if r := recover(); r != nil {
					errs[idx] = fmt.Errorf("panic during extraction: %v", r)
				}
				wg.Done()
			}()

			results[idx], errs[idx] = p.extractWithConfig(htmlBytes, config)
		}(i, content)
	}

	wg.Wait()
	return collectResults(results, errs, nil)
}

// ExtractBatchWithContext extracts content from multiple HTML byte slices concurrently with context support.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// If the context is cancelled, pending extractions are skipped and the BatchResult.Cancelled count is incremented.
func (p *Processor) ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte) *BatchResult {
	br := &BatchResult{
		Results: make([]*Result, len(htmlContents)),
		Errors:  make([]error, len(htmlContents)),
	}

	if p == nil || p.closed.Load() {
		for i := range htmlContents {
			br.Errors[i] = ErrProcessorClosed
		}
		br.Failed = len(htmlContents)
		return br
	}

	if len(htmlContents) == 0 {
		return br
	}

	config := p.getExtractConfig()

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, p.config.WorkerPoolSize)

	for i, content := range htmlContents {
		// Check context before starting each job
		select {
		case <-ctx.Done():
			mu.Lock()
			br.Cancelled++
			br.Errors[i] = ctx.Err()
			mu.Unlock()
			continue
		default:
		}

		wg.Add(1)
		go func(idx int, htmlBytes []byte) {
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					br.Errors[idx] = fmt.Errorf("panic during extraction: %v", r)
					br.Failed++
					mu.Unlock()
				}
				wg.Done()
			}()

			// Check context before acquiring semaphore
			select {
			case <-ctx.Done():
				mu.Lock()
				br.Cancelled++
				br.Errors[idx] = ctx.Err()
				mu.Unlock()
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			// Check context before processing
			select {
			case <-ctx.Done():
				mu.Lock()
				br.Cancelled++
				br.Errors[idx] = ctx.Err()
				mu.Unlock()
				return
			default:
			}

			result, err := p.extractWithConfig(htmlBytes, config)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				br.Errors[idx] = err
				br.Failed++
			} else {
				br.Results[idx] = result
				br.Success++
			}
		}(i, content)
	}

	wg.Wait()
	return br
}

// ExtractBatchFiles extracts content from multiple HTML files concurrently.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// Returns a slice of results and a combined error if any extractions failed.
func (p *Processor) ExtractBatchFiles(filePaths []string) ([]*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(filePaths) == 0 {
		return []*Result{}, nil
	}

	config := p.getExtractConfig()

	results := make([]*Result, len(filePaths))
	errs := make([]error, len(filePaths))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, path := range filePaths {
		// Acquire semaphore BEFORE creating goroutine to limit concurrent goroutines
		sem <- struct{}{}

		wg.Add(1)
		go func(idx int, filePath string) {
			defer func() {
				<-sem
				if r := recover(); r != nil {
					errs[idx] = fmt.Errorf("panic during file extraction: %v", r)
				}
				wg.Done()
			}()

			results[idx], errs[idx] = p.extractFromFileWithConfig(filePath, config)
		}(i, path)
	}

	wg.Wait()
	return collectResults(results, errs, filePaths)
}

// ExtractBatchFilesWithContext extracts content from multiple HTML files concurrently with context support.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// If the context is cancelled, pending extractions are skipped and the BatchResult.Cancelled count is incremented.
func (p *Processor) ExtractBatchFilesWithContext(ctx context.Context, filePaths []string) *BatchResult {
	br := &BatchResult{
		Results: make([]*Result, len(filePaths)),
		Errors:  make([]error, len(filePaths)),
	}

	if p == nil || p.closed.Load() {
		for i := range filePaths {
			br.Errors[i] = ErrProcessorClosed
		}
		br.Failed = len(filePaths)
		return br
	}

	if len(filePaths) == 0 {
		return br
	}

	config := p.getExtractConfig()

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, p.config.WorkerPoolSize)

	for i, path := range filePaths {
		// Check context before starting each job
		select {
		case <-ctx.Done():
			mu.Lock()
			br.Cancelled++
			br.Errors[i] = ctx.Err()
			mu.Unlock()
			continue
		default:
		}

		wg.Add(1)
		go func(idx int, filePath string) {
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					br.Errors[idx] = fmt.Errorf("panic during file extraction: %v", r)
					br.Failed++
					mu.Unlock()
				}
				wg.Done()
			}()

			// Check context before acquiring semaphore
			select {
			case <-ctx.Done():
				mu.Lock()
				br.Cancelled++
				br.Errors[idx] = ctx.Err()
				mu.Unlock()
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			// Check context before processing
			select {
			case <-ctx.Done():
				mu.Lock()
				br.Cancelled++
				br.Errors[idx] = ctx.Err()
				mu.Unlock()
				return
			default:
			}

			result, err := p.extractFromFileWithConfig(filePath, config)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				br.Errors[idx] = err
				br.Failed++
			} else {
				br.Results[idx] = result
				br.Success++
			}
		}(i, path)
	}

	wg.Wait()
	return br
}

// extractWithConfig extracts content using the provided config (internal helper for batch operations).
func (p *Processor) extractWithConfig(htmlBytes []byte, config ExtractConfig) (*Result, error) {
	// Use the internal extraction logic directly with the provided config
	return p.extractInternal(htmlBytes, config)
}

// extractFromFileWithConfig extracts content from file using the provided config (internal helper for batch operations).
func (p *Processor) extractFromFileWithConfig(filePath string, config ExtractConfig) (*Result, error) {
	// Validate file path
	if filePath == "" {
		return nil, fmt.Errorf("%w: empty file path", ErrInvalidFilePath)
	}

	cleanPath := filepathClean(filePath)
	if stringsContains(cleanPath, "..") {
		return nil, fmt.Errorf("%w: path traversal detected: %s", ErrInvalidFilePath, cleanPath)
	}

	data, readErr := readFile(cleanPath)
	if readErr != nil {
		if osIsNotExist(readErr) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, cleanPath)
		}
		return nil, fmt.Errorf("read file %q: %w", cleanPath, readErr)
	}

	return p.extractInternal(data, config)
}

// extractInternal performs the actual extraction with the given config.
func (p *Processor) extractInternal(htmlBytes []byte, config ExtractConfig) (*Result, error) {
	if p == nil || p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(htmlBytes) > p.config.MaxInputSize {
		return nil, fmt.Errorf("%w: size=%d, max=%d", ErrInputTooLarge, len(htmlBytes), p.config.MaxInputSize)
	}

	utf8String, _, convErr := internal.DetectAndConvertToUTF8String(htmlBytes, config.Encoding)
	if convErr != nil {
		return nil, fmt.Errorf("encoding detection failed: %w", convErr)
	}

	return p.processContent(utf8String, config)
}
