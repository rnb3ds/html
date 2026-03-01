package html

import (
	"context"
	"sync"
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
// This is a convenience function that creates a temporary processor for one-time batch extraction.
// For repeated batch operations, use a persistent Processor instance.
func ExtractBatch(htmlContents [][]byte, configs ...ExtractConfig) ([]*Result, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractBatch(htmlContents, configs...)
}

// ExtractBatchWithContext extracts content from multiple HTML byte slices concurrently with context support.
// This is a convenience function that creates a temporary processor for one-time batch extraction.
// For repeated batch operations, use a persistent Processor instance.
func ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte, configs ...ExtractConfig) *BatchResult {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractBatchWithContext(ctx, htmlContents, configs...)
}

// ExtractBatchFiles extracts content from multiple HTML files concurrently.
// This is a convenience function that creates a temporary processor for one-time batch extraction.
// For repeated batch operations, use a persistent Processor instance.
func ExtractBatchFiles(filePaths []string, configs ...ExtractConfig) ([]*Result, error) {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractBatchFiles(filePaths, configs...)
}

// ExtractBatchFilesWithContext extracts content from multiple HTML files concurrently with context support.
// This is a convenience function that creates a temporary processor for one-time batch extraction.
// For repeated batch operations, use a persistent Processor instance.
func ExtractBatchFilesWithContext(ctx context.Context, filePaths []string, configs ...ExtractConfig) *BatchResult {
	processor, _ := New()
	defer processor.Close()
	return processor.ExtractBatchFilesWithContext(ctx, filePaths, configs...)
}

// ExtractBatch extracts content from multiple HTML byte slices concurrently.
func (p *Processor) ExtractBatch(htmlContents [][]byte, configs ...ExtractConfig) ([]*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(htmlContents) == 0 {
		return []*Result{}, nil
	}

	config := resolveExtractConfig(configs...)

	results := make([]*Result, len(htmlContents))
	errs := make([]error, len(htmlContents))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, content := range htmlContents {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, htmlBytes []byte) {
			defer wg.Done()
			defer func() { <-sem }()

			results[idx], errs[idx] = p.Extract(htmlBytes, config)
		}(i, content)
	}

	wg.Wait()
	return collectResults(results, errs, nil)
}

// ExtractBatchWithContext extracts content from multiple HTML byte slices concurrently with context support.
// If the context is cancelled, pending extractions are skipped and the BatchResult.Cancelled count is incremented.
func (p *Processor) ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte, configs ...ExtractConfig) *BatchResult {
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

	config := resolveExtractConfig(configs...)

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
			defer wg.Done()

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

			result, err := p.Extract(htmlBytes, config)
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
func (p *Processor) ExtractBatchFiles(filePaths []string, configs ...ExtractConfig) ([]*Result, error) {
	if p == nil {
		return nil, ErrProcessorClosed
	}
	if p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(filePaths) == 0 {
		return []*Result{}, nil
	}

	config := resolveExtractConfig(configs...)

	results := make([]*Result, len(filePaths))
	errs := make([]error, len(filePaths))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, path := range filePaths {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, filePath string) {
			defer wg.Done()
			defer func() { <-sem }()

			results[idx], errs[idx] = p.ExtractFromFile(filePath, config)
		}(i, path)
	}

	wg.Wait()
	return collectResults(results, errs, filePaths)
}

// ExtractBatchFilesWithContext extracts content from multiple HTML files concurrently with context support.
// If the context is cancelled, pending extractions are skipped and the BatchResult.Cancelled count is incremented.
func (p *Processor) ExtractBatchFilesWithContext(ctx context.Context, filePaths []string, configs ...ExtractConfig) *BatchResult {
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

	config := resolveExtractConfig(configs...)

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
			defer wg.Done()

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

			result, err := p.ExtractFromFile(filePath, config)
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
