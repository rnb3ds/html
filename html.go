// Package html provides HTML content extraction with automatic encoding detection.
//
// This library extracts clean text, links, images, videos, and audio from HTML documents
// with automatic character encoding detection (supporting 15+ encodings including UTF-8,
// Windows-1252, GBK, Shift_JIS, etc.).
//
// Basic usage:
//
//	// Simple extraction
//	result, err := html.Extract(htmlBytes)
//	fmt.Println(result.Text)
//
//	// With processor for repeated extractions
//	processor, _ := html.New(html.WithCache(1000, time.Hour))
//	defer processor.Close()
//	result, _ := processor.Extract(htmlBytes)
package html
