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

// extractFunc is a function type for extracting content from a single input.
type extractFunc func(config ExtractConfig) (*Result, error)

// ExtractBatch extracts content from multiple HTML byte slices concurrently.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// Returns a slice of results and a combined error if any extractions failed.
func (p *Processor) ExtractBatch(htmlContents [][]byte) ([]*Result, error) {
	if p == nil || p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(htmlContents) == 0 {
		return []*Result{}, nil
	}

	config := p.getExtractConfig()
	extractors := make([]extractFunc, len(htmlContents))
	for i, content := range htmlContents {
		extractors[i] = p.extractorForBytes(content)
	}

	return p.runBatch(extractors, config, nil)
}

// ExtractBatchWithContext extracts content from multiple HTML byte slices concurrently with context support.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// If the context is cancelled, pending extractions are skipped and the BatchResult.Cancelled count is incremented.
func (p *Processor) ExtractBatchWithContext(ctx context.Context, htmlContents [][]byte) *BatchResult {
	if p == nil || p.closed.Load() {
		return p.closedBatchResult(len(htmlContents))
	}

	if len(htmlContents) == 0 {
		return &BatchResult{Results: []*Result{}, Errors: []error{}}
	}

	config := p.getExtractConfig()
	extractors := make([]extractFunc, len(htmlContents))
	for i, content := range htmlContents {
		extractors[i] = p.extractorForBytes(content)
	}

	return p.runBatchWithContext(ctx, extractors, config)
}

// ExtractBatchFiles extracts content from multiple HTML files concurrently.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// Returns a slice of results and a combined error if any extractions failed.
func (p *Processor) ExtractBatchFiles(filePaths []string) ([]*Result, error) {
	if p == nil || p.closed.Load() {
		return nil, ErrProcessorClosed
	}

	if len(filePaths) == 0 {
		return []*Result{}, nil
	}

	config := p.getExtractConfig()
	extractors := make([]extractFunc, len(filePaths))
	for i, path := range filePaths {
		extractors[i] = p.extractorForFile(path)
	}

	return p.runBatch(extractors, config, filePaths)
}

// ExtractBatchFilesWithContext extracts content from multiple HTML files concurrently with context support.
// The concurrency level is controlled by the WorkerPoolSize configuration (default: 4).
// Each extraction is performed independently with automatic encoding detection.
// If the context is cancelled, pending extractions are skipped and the BatchResult.Cancelled count is incremented.
func (p *Processor) ExtractBatchFilesWithContext(ctx context.Context, filePaths []string) *BatchResult {
	if p == nil || p.closed.Load() {
		return p.closedBatchResult(len(filePaths))
	}

	if len(filePaths) == 0 {
		return &BatchResult{Results: []*Result{}, Errors: []error{}}
	}

	config := p.getExtractConfig()
	extractors := make([]extractFunc, len(filePaths))
	for i, path := range filePaths {
		extractors[i] = p.extractorForFile(path)
	}

	return p.runBatchWithContext(ctx, extractors, config)
}

// extractorForBytes creates an extractFunc for byte slice input.
func (p *Processor) extractorForBytes(htmlBytes []byte) extractFunc {
	return func(config ExtractConfig) (*Result, error) {
		return p.extractWithConfig(htmlBytes, config)
	}
}

// extractorForFile creates an extractFunc for file path input.
func (p *Processor) extractorForFile(filePath string) extractFunc {
	return func(config ExtractConfig) (*Result, error) {
		return p.extractFromFileWithConfig(filePath, config)
	}
}

// runBatch executes a batch of extractions concurrently without context.
func (p *Processor) runBatch(extractors []extractFunc, config ExtractConfig, names []string) ([]*Result, error) {
	results := make([]*Result, len(extractors))
	errs := make([]error, len(extractors))
	sem := make(chan struct{}, p.config.WorkerPoolSize)
	var wg sync.WaitGroup

	for i, extractor := range extractors {
		// Acquire semaphore BEFORE creating goroutine to limit concurrent goroutines
		sem <- struct{}{}

		wg.Add(1)
		go func(idx int, extract extractFunc) {
			defer func() {
				<-sem
				if r := recover(); r != nil {
					errs[idx] = fmt.Errorf("panic during extraction: %v", r)
				}
				wg.Done()
			}()

			results[idx], errs[idx] = extract(config)
		}(i, extractor)
	}

	wg.Wait()
	return collectResults(results, errs, names)
}

// runBatchWithContext executes a batch of extractions concurrently with context support.
func (p *Processor) runBatchWithContext(ctx context.Context, extractors []extractFunc, config ExtractConfig) *BatchResult {
	br := &BatchResult{
		Results: make([]*Result, len(extractors)),
		Errors:  make([]error, len(extractors)),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, p.config.WorkerPoolSize)

	for i, extractor := range extractors {
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
		go func(idx int, extract extractFunc) {
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

			result, err := extract(config)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				br.Errors[idx] = err
				br.Failed++
			} else {
				br.Results[idx] = result
				br.Success++
			}
		}(i, extractor)
	}

	wg.Wait()
	return br
}

// closedBatchResult creates a BatchResult for a closed processor.
func (p *Processor) closedBatchResult(count int) *BatchResult {
	br := &BatchResult{
		Results: make([]*Result, count),
		Errors:  make([]error, count),
		Failed:  count,
	}
	for i := 0; i < count; i++ {
		br.Errors[i] = ErrProcessorClosed
	}
	return br
}

// extractWithConfig extracts content using the provided config (internal helper for batch operations).
func (p *Processor) extractWithConfig(htmlBytes []byte, config ExtractConfig) (*Result, error) {
	// Use the internal extraction logic directly with the provided config
	return p.extractInternal(htmlBytes, config)
}

// extractFromFileWithConfig extracts content from file using the provided config (internal helper for batch operations).
func (p *Processor) extractFromFileWithConfig(filePath string, config ExtractConfig) (*Result, error) {
	data, err := p.validateAndReadFile(filePath)
	if err != nil {
		return nil, err
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
