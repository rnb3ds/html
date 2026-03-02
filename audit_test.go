package html_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cybergodev/html"
)

// TestAuditLoggingEnabled tests that audit logging captures security events
func TestAuditLoggingEnabled(t *testing.T) {
	t.Parallel()

	auditConfig := html.DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.IncludeRawValues = true

	config := html.DefaultConfig()
	config.Audit = auditConfig

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Process HTML with XSS payloads
	xssHTML := `<html><body>
		<script>alert('XSS')</script>
		<div onclick="alert('XSS')">Content</div>
		<a href="javascript:alert('XSS')">Link</a>
		<p>Safe content</p>
	</body></html>`

	_, err = p.Extract([]byte(xssHTML))
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	entries := p.GetAuditLog()
	if len(entries) == 0 {
		t.Error("Expected audit log entries, got none")
	}

	// Verify we captured blocked tags and attributes
	foundBlockedTag := false
	foundBlockedAttr := false
	foundBlockedURL := false

	for _, entry := range entries {
		switch entry.EventType {
		case html.AuditEventBlockedTag:
			foundBlockedTag = true
			if entry.Tag != "script" {
				t.Errorf("Expected blocked tag 'script', got '%s'", entry.Tag)
			}
		case html.AuditEventBlockedAttr:
			foundBlockedAttr = true
			if entry.Attribute != "onclick" {
				t.Errorf("Expected blocked attribute 'onclick', got '%s'", entry.Attribute)
			}
		case html.AuditEventBlockedURL:
			foundBlockedURL = true
		}
	}

	if !foundBlockedTag {
		t.Error("Expected blocked tag event to be logged")
	}
	if !foundBlockedAttr {
		t.Error("Expected blocked attribute event to be logged")
	}
	if !foundBlockedURL {
		t.Error("Expected blocked URL event to be logged")
	}
}

// TestAuditLoggingDisabled tests that audit logging can be disabled
func TestAuditLoggingDisabled(t *testing.T) {
	t.Parallel()

	auditConfig := html.DefaultAuditConfig()
	auditConfig.Enabled = false

	config := html.DefaultConfig()
	config.Audit = auditConfig

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	xssHTML := `<html><body><script>alert('XSS')</script><p>Content</p></body></html>`

	_, err = p.Extract([]byte(xssHTML))
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	entries := p.GetAuditLog()
	if len(entries) != 0 {
		t.Errorf("Expected no audit log entries when disabled, got %d", len(entries))
	}
}

// TestAuditInputViolation tests logging of input size violations
func TestAuditInputViolation(t *testing.T) {
	t.Parallel()

	auditConfig := html.DefaultAuditConfig()
	auditConfig.Enabled = true

	config := html.DefaultConfig()
	config.MaxInputSize = 1000 // 1KB limit
	config.Audit = auditConfig

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	largeHTML := strings.Repeat("<div>test content</div>", 100) // ~2KB

	_, err = p.Extract([]byte(largeHTML))
	if err == nil {
		t.Fatal("Expected error for oversized input")
	}

	entries := p.GetAuditLog()
	if len(entries) == 0 {
		t.Fatal("Expected audit log entries for input violation")
	}

	found := false
	for _, entry := range entries {
		if entry.EventType == html.AuditEventInputViolation {
			found = true
			if entry.InputSize <= entry.MaxSize {
				t.Error("Input size should exceed max size in log entry")
			}
		}
	}

	if !found {
		t.Error("Expected input violation event to be logged")
	}
}

// TestAuditPathTraversal tests logging of path traversal attempts
func TestAuditPathTraversal(t *testing.T) {
	t.Parallel()

	auditConfig := html.DefaultAuditConfig()
	auditConfig.Enabled = true

	config := html.DefaultConfig()
	config.Audit = auditConfig

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	_, err = p.ExtractFromFile("../../../etc/passwd")
	if err == nil {
		t.Fatal("Expected error for path traversal attempt")
	}

	entries := p.GetAuditLog()
	if len(entries) == 0 {
		t.Fatal("Expected audit log entries for path traversal")
	}

	found := false
	for _, entry := range entries {
		if entry.EventType == html.AuditEventPathTraversal {
			found = true
			if entry.Path == "" {
				t.Error("Path should be recorded in path traversal event")
			}
		}
	}

	if !found {
		t.Error("Expected path traversal event to be logged")
	}
}

// TestAuditCollector tests the AuditCollector directly
func TestAuditCollector(t *testing.T) {
	t.Parallel()

	config := html.HighSecurityAuditConfig()
	collector := html.NewAuditCollector(config)
	defer collector.Close()

	collector.RecordBlockedTag("script")
	collector.RecordBlockedAttr("onclick", "alert(1)")
	collector.RecordBlockedURL("javascript:alert(1)", "javascript scheme")

	entries := collector.GetEntries()
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	// Verify timestamps are set
	for _, entry := range entries {
		if entry.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}
	}

	// Clear and verify
	collector.Clear()
	entries = collector.GetEntries()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}

// TestAuditRawValueTruncation tests that raw values are truncated
func TestAuditRawValueTruncation(t *testing.T) {
	t.Parallel()

	auditConfig := html.DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.IncludeRawValues = true
	auditConfig.MaxRawValueLength = 50

	config := html.DefaultConfig()
	config.Audit = auditConfig

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Create a very long attribute value
	longValue := strings.Repeat("A", 200)
	xssHTML := `<html><body><div onclick="` + longValue + `">Content</div></body></html>`

	_, err = p.Extract([]byte(xssHTML))
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	entries := p.GetAuditLog()
	for _, entry := range entries {
		if entry.EventType == html.AuditEventBlockedAttr {
			if len(entry.RawValue) > 53 { // 50 + "..."
				t.Errorf("Raw value should be truncated, got length %d", len(entry.RawValue))
			}
		}
	}
}

// TestAuditExcludeRawValues tests that raw values can be excluded
func TestAuditExcludeRawValues(t *testing.T) {
	t.Parallel()

	auditConfig := html.DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.IncludeRawValues = false

	config := html.DefaultConfig()
	config.Audit = auditConfig

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	xssHTML := `<html><body><div onclick="alert('XSS')">Content</div></body></html>`

	_, err = p.Extract([]byte(xssHTML))
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	entries := p.GetAuditLog()
	for _, entry := range entries {
		if entry.EventType == html.AuditEventBlockedAttr {
			if entry.RawValue != "" {
				t.Errorf("Raw value should be empty when excluded, got '%s'", entry.RawValue)
			}
		}
	}
}

// TestLoggerAuditSink tests the logger audit sink
func TestLoggerAuditSink(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	sink := html.NewLoggerAuditSinkWithWriter(&buf)
	defer sink.Close()

	entry := html.AuditEntry{
		Timestamp: time.Now(),
		EventType: html.AuditEventBlockedTag,
		Level:     html.AuditLevelWarning,
		Message:   "Test message",
		Tag:       "script",
	}

	sink.Write(entry)

	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

// TestWriterAuditSink tests the writer audit sink
func TestWriterAuditSink(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	sink := html.NewWriterAuditSink(&buf)

	entry := html.AuditEntry{
		Timestamp: time.Now(),
		EventType: html.AuditEventBlockedTag,
		Level:     html.AuditLevelWarning,
		Message:   "Test message",
	}

	sink.Write(entry)

	output := buf.String()
	if output == "" {
		t.Error("Expected output from writer sink")
	}

	// Verify it's valid JSON
	var decoded html.AuditEntry
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Errorf("Output should be valid JSON: %v", err)
	}
}

// TestChannelAuditSink tests the channel audit sink
func TestChannelAuditSink(t *testing.T) {
	t.Parallel()

	sink := html.NewChannelAuditSink(10)
	defer sink.Close()

	entry := html.AuditEntry{
		Timestamp: time.Now(),
		EventType: html.AuditEventBlockedTag,
		Level:     html.AuditLevelWarning,
		Message:   "Test message",
	}

	sink.Write(entry)

	select {
	case received := <-sink.Channel():
		if received.Message != entry.Message {
			t.Errorf("Expected message '%s', got '%s'", entry.Message, received.Message)
		}
	case <-time.After(time.Second):
		t.Error("Expected to receive entry from channel")
	}
}

// TestMultiSink tests the multi sink
func TestMultiSink(t *testing.T) {
	t.Parallel()

	var buf1, buf2 bytes.Buffer
	sink1 := html.NewWriterAuditSink(&buf1)
	sink2 := html.NewWriterAuditSink(&buf2)

	multi := html.NewMultiSink(sink1, sink2)
	defer multi.Close()

	entry := html.AuditEntry{
		Timestamp: time.Now(),
		EventType: html.AuditEventBlockedTag,
		Message:   "Test message",
	}

	multi.Write(entry)

	if buf1.String() == "" {
		t.Error("Expected sink1 to receive entry")
	}
	if buf2.String() == "" {
		t.Error("Expected sink2 to receive entry")
	}
}

// TestLevelFilteredSink tests the level filtered sink
func TestLevelFilteredSink(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writerSink := html.NewWriterAuditSink(&buf)
	filtered := html.NewLevelFilteredSink(writerSink, html.AuditLevelWarning)
	defer filtered.Close()

	// Info level should be filtered out
	infoEntry := html.AuditEntry{
		Timestamp: time.Now(),
		EventType: html.AuditEventEncodingIssue,
		Level:     html.AuditLevelInfo,
		Message:   "Info message",
	}
	filtered.Write(infoEntry)

	if buf.String() != "" {
		t.Error("Info level should be filtered out")
	}

	// Warning level should pass
	warningEntry := html.AuditEntry{
		Timestamp: time.Now(),
		EventType: html.AuditEventBlockedTag,
		Level:     html.AuditLevelWarning,
		Message:   "Warning message",
	}
	filtered.Write(warningEntry)

	if buf.String() == "" {
		t.Error("Warning level should pass filter")
	}
}

// TestFilteredSink tests the generic filtered sink with custom filter function
func TestFilteredSink(t *testing.T) {
	t.Parallel()

	t.Run("filter blocks specific event types", func(t *testing.T) {
		var buf bytes.Buffer
		writerSink := html.NewWriterAuditSink(&buf)

		// Filter that only allows blocked tag events
		filter := func(entry html.AuditEntry) bool {
			return entry.EventType == html.AuditEventBlockedTag
		}

		filtered := html.NewFilteredSink(writerSink, filter)
		defer filtered.Close()

		// Entry that should pass filter
		blockedTagEntry := html.AuditEntry{
			Timestamp: time.Now(),
			EventType: html.AuditEventBlockedTag,
			Level:     html.AuditLevelWarning,
			Message:   "Blocked tag",
		}
		filtered.Write(blockedTagEntry)

		if buf.String() == "" {
			t.Error("BlockedTag event should pass filter")
		}

		buf.Reset()

		// Entry that should be filtered out
		encodingEntry := html.AuditEntry{
			Timestamp: time.Now(),
			EventType: html.AuditEventEncodingIssue,
			Level:     html.AuditLevelInfo,
			Message:   "Encoding issue",
		}
		filtered.Write(encodingEntry)

		if buf.String() != "" {
			t.Error("EncodingIssue event should be filtered out")
		}
	})

	t.Run("filter allows all when filter is nil", func(t *testing.T) {
		var buf bytes.Buffer
		writerSink := html.NewWriterAuditSink(&buf)

		filtered := html.NewFilteredSink(writerSink, nil)
		defer filtered.Close()

		entry := html.AuditEntry{
			Timestamp: time.Now(),
			EventType: html.AuditEventBlockedTag,
			Message:   "Test message",
		}
		filtered.Write(entry)

		if buf.String() == "" {
			t.Error("Nil filter should allow all entries")
		}
	})

	t.Run("filter by level", func(t *testing.T) {
		var buf bytes.Buffer
		writerSink := html.NewWriterAuditSink(&buf)

		// Filter that only allows critical level
		filter := func(entry html.AuditEntry) bool {
			return entry.Level == html.AuditLevelCritical
		}

		filtered := html.NewFilteredSink(writerSink, filter)
		defer filtered.Close()

		// Warning should be filtered out
		warningEntry := html.AuditEntry{
			Timestamp: time.Now(),
			EventType: html.AuditEventBlockedTag,
			Level:     html.AuditLevelWarning,
			Message:   "Warning",
		}
		filtered.Write(warningEntry)

		if buf.String() != "" {
			t.Error("Warning level should be filtered out")
		}

		// Critical should pass
		criticalEntry := html.AuditEntry{
			Timestamp: time.Now(),
			EventType: html.AuditEventInputViolation,
			Level:     html.AuditLevelCritical,
			Message:   "Critical",
		}
		filtered.Write(criticalEntry)

		if buf.String() == "" {
			t.Error("Critical level should pass filter")
		}
	})

	t.Run("close propagates to underlying sink", func(t *testing.T) {
		var buf bytes.Buffer
		writerSink := html.NewWriterAuditSink(&buf)
		filtered := html.NewFilteredSink(writerSink, nil)

		// Close should not return error
		if err := filtered.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	t.Run("nil sink handles gracefully", func(t *testing.T) {
		filtered := html.NewFilteredSink(nil, func(entry html.AuditEntry) bool {
			return true
		})

		// Write should not panic with nil sink
		entry := html.AuditEntry{
			Timestamp: time.Now(),
			EventType: html.AuditEventBlockedTag,
			Message:   "Test",
		}
		filtered.Write(entry)

		// Close should handle nil sink
		if err := filtered.Close(); err != nil {
			t.Errorf("Close() with nil sink returned error: %v", err)
		}
	})

	t.Run("nil filtered sink handles gracefully", func(t *testing.T) {
		var filtered *html.FilteredSink

		// Write should not panic with nil filtered sink
		entry := html.AuditEntry{
			Timestamp: time.Now(),
			EventType: html.AuditEventBlockedTag,
			Message:   "Test",
		}
		filtered.Write(entry)

		// Close should handle nil filtered sink
		if err := filtered.Close(); err != nil {
			t.Errorf("Close() with nil filtered sink returned error: %v", err)
		}
	})
}

// TestHighSecurityConfigAudit tests that HighSecurityConfig has audit enabled
func TestHighSecurityConfigAudit(t *testing.T) {
	t.Parallel()

	config := html.HighSecurityConfig()

	if !config.Audit.Enabled {
		t.Error("HighSecurityConfig should have audit enabled")
	}
	if !config.Audit.LogBlockedTags {
		t.Error("HighSecurityConfig should log blocked tags")
	}
	if !config.Audit.LogBlockedAttrs {
		t.Error("HighSecurityConfig should log blocked attributes")
	}
	if !config.Audit.LogBlockedURLs {
		t.Error("HighSecurityConfig should log blocked URLs")
	}
	if !config.Audit.IncludeRawValues {
		t.Error("HighSecurityConfig should include raw values for forensics")
	}
}

// TestAuditEntryJSON tests JSON serialization of audit entries
func TestAuditEntryJSON(t *testing.T) {
	t.Parallel()

	entry := html.AuditEntry{
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		EventType: html.AuditEventBlockedTag,
		Level:     html.AuditLevelWarning,
		Message:   "Blocked dangerous HTML tag",
		Tag:       "script",
		Metadata: map[string]any{
			"source": "test",
		},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal audit entry: %v", err)
	}

	var decoded html.AuditEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal audit entry: %v", err)
	}

	if decoded.EventType != entry.EventType {
		t.Errorf("Expected EventType %s, got %s", entry.EventType, decoded.EventType)
	}
	if decoded.Tag != entry.Tag {
		t.Errorf("Expected Tag %s, got %s", entry.Tag, decoded.Tag)
	}
}

// TestAuditWithCustomSink tests using a custom sink
func TestAuditWithCustomSink(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	customSink := html.NewWriterAuditSink(&buf)

	auditConfig := html.DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.Sink = customSink

	config := html.DefaultConfig()
	config.Audit = auditConfig

	p, err := html.New(config)
	if err != nil {
		t.Fatal(err)
	}

	xssHTML := `<html><body><script>alert('XSS')</script><p>Content</p></body></html>`

	_, err = p.Extract([]byte(xssHTML))
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	// Close processor first to ensure all async audit writes complete
	// This waits for all goroutines to finish via WaitGroup
	p.Close()

	output := buf.String()
	if output == "" {
		t.Error("Expected custom sink to receive audit entries")
	}
}

// BenchmarkAuditLogging benchmarks the overhead of audit logging
func BenchmarkAuditLogging(b *testing.B) {
	b.Run("disabled", func(b *testing.B) {
		config := html.DefaultConfig()
		config.Audit.Enabled = false
		p, _ := html.New(config)
		defer p.Close()

		htmlContent := []byte(`<html><body><h1>Test</h1><p>Content</p></body></html>`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			p.Extract(htmlContent)
		}
	})

	b.Run("enabled", func(b *testing.B) {
		config := html.DefaultConfig()
		config.Audit.Enabled = true
		p, _ := html.New(config)
		defer p.Close()

		htmlContent := []byte(`<html><body><h1>Test</h1><p>Content</p></body></html>`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			p.Extract(htmlContent)
			p.ClearAuditLog()
		}
	})

	b.Run("enabled_with_xss", func(b *testing.B) {
		config := html.DefaultConfig()
		config.Audit.Enabled = true
		p, _ := html.New(config)
		defer p.Close()

		htmlContent := []byte(`<html><body><script>alert(1)</script><p>Content</p></body></html>`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			p.Extract(htmlContent)
			p.ClearAuditLog()
		}
	})
}

// TestChannelAuditSinkConcurrentClose tests that concurrent Close calls are safe
func TestChannelAuditSinkConcurrentClose(t *testing.T) {
	t.Parallel()

	const numGoroutines = 100

	for i := 0; i < 10; i++ {
		sink := html.NewChannelAuditSink(10)

		var wg sync.WaitGroup
		var panicCount int64

		for j := 0; j < numGoroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						// Concurrent close should not panic
						panicCount++
					}
				}()
				sink.Close()
			}()
		}

		wg.Wait()

		if panicCount > 0 {
			t.Errorf("Concurrent Close caused %d panics", panicCount)
		}
	}
}

// TestAuditCollectorWait tests the Wait method for async sink writes
func TestAuditCollectorWait(t *testing.T) {
	t.Parallel()

	t.Run("Wait blocks until async writes complete", func(t *testing.T) {
		var buf bytes.Buffer
		sink := html.NewWriterAuditSink(&buf)

		config := html.HighSecurityAuditConfig()
		config.Sink = sink
		collector := html.NewAuditCollector(config)
		defer collector.Close()

		// Record multiple entries
		for i := 0; i < 10; i++ {
			collector.RecordBlockedTag("script")
		}

		// Wait for all async writes
		collector.Wait()

		// All entries should be written
		output := buf.String()
		if output == "" {
			t.Error("Expected audit output after Wait()")
		}

		// Count the number of JSON entries
		entryCount := strings.Count(output, "\"event_type\"")
		if entryCount != 10 {
			t.Errorf("Expected 10 entries, got %d", entryCount)
		}
	})

	t.Run("Wait on nil collector is safe", func(t *testing.T) {
		var collector *html.AuditCollector
		// Should not panic
		collector.Wait()
	})

	t.Run("Wait with disabled audit is safe", func(t *testing.T) {
		config := html.DefaultAuditConfig() // Disabled by default
		collector := html.NewAuditCollector(config)
		defer collector.Close()

		collector.RecordBlockedTag("script")
		collector.Wait() // Should complete immediately
	})

	t.Run("Wait allows multiple calls", func(t *testing.T) {
		config := html.HighSecurityAuditConfig()
		collector := html.NewAuditCollector(config)
		defer collector.Close()

		collector.RecordBlockedTag("script")
		collector.Wait()
		collector.Wait() // Second call should be safe
		collector.Wait() // Third call should be safe
	})
}

// TestRecordEncodingIssue tests the RecordEncodingIssue method
func TestRecordEncodingIssue(t *testing.T) {
	t.Parallel()

	t.Run("records encoding issue event", func(t *testing.T) {
		config := html.DefaultAuditConfig()
		config.Enabled = true
		config.LogEncodingIssues = true
		collector := html.NewAuditCollector(config)
		defer collector.Close()

		collector.RecordEncodingIssue("windows-1252", "invalid byte sequence")

		entries := collector.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}

		if entries[0].EventType != html.AuditEventEncodingIssue {
			t.Errorf("Expected EventType %s, got %s", html.AuditEventEncodingIssue, entries[0].EventType)
		}
		if entries[0].Level != html.AuditLevelInfo {
			t.Errorf("Expected Level %s, got %s", html.AuditLevelInfo, entries[0].Level)
		}
		if entries[0].Message != "invalid byte sequence" {
			t.Errorf("Expected message 'invalid byte sequence', got '%s'", entries[0].Message)
		}
		if entries[0].Metadata["encoding"] != "windows-1252" {
			t.Errorf("Expected encoding metadata, got %v", entries[0].Metadata["encoding"])
		}
	})

	t.Run("nil collector handles gracefully", func(t *testing.T) {
		var collector *html.AuditCollector
		// Should not panic
		collector.RecordEncodingIssue("utf-8", "test")
	})

	t.Run("disabled audit does not record", func(t *testing.T) {
		config := html.DefaultAuditConfig()
		config.Enabled = false
		collector := html.NewAuditCollector(config)
		defer collector.Close()

		collector.RecordEncodingIssue("utf-8", "test")

		entries := collector.GetEntries()
		if len(entries) != 0 {
			t.Errorf("Expected 0 entries when disabled, got %d", len(entries))
		}
	})

	t.Run("LogEncodingIssues disabled does not record", func(t *testing.T) {
		config := html.DefaultAuditConfig()
		config.Enabled = true
		config.LogEncodingIssues = false
		collector := html.NewAuditCollector(config)
		defer collector.Close()

		collector.RecordEncodingIssue("utf-8", "test")

		entries := collector.GetEntries()
		if len(entries) != 0 {
			t.Errorf("Expected 0 entries when LogEncodingIssues disabled, got %d", len(entries))
		}
	})

	t.Run("records with various encodings", func(t *testing.T) {
		config := html.DefaultAuditConfig()
		config.Enabled = true
		config.LogEncodingIssues = true
		collector := html.NewAuditCollector(config)
		defer collector.Close()

		encodings := []string{"utf-8", "windows-1252", "iso-8859-1", "shift_jis", "gbk"}
		for _, enc := range encodings {
			collector.RecordEncodingIssue(enc, "detection failed")
		}

		entries := collector.GetEntries()
		if len(entries) != len(encodings) {
			t.Errorf("Expected %d entries, got %d", len(encodings), len(entries))
		}
	})
}

// TestClearAuditLog tests the ClearAuditLog method on Processor
func TestClearAuditLog(t *testing.T) {
	t.Parallel()

	t.Run("ClearAuditLog clears processor audit entries", func(t *testing.T) {
		auditConfig := html.DefaultAuditConfig()
		auditConfig.Enabled = true

		config := html.DefaultConfig()
		config.Audit = auditConfig
		p, err := html.New(config)
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		// Process XSS HTML to generate audit entries
		xssHTML := `<html><body><script>alert('XSS')</script><p>Content</p></body></html>`
		_, err = p.Extract([]byte(xssHTML))
		if err != nil {
			t.Fatalf("Extract() failed: %v", err)
		}

		// Verify entries exist
		entries := p.GetAuditLog()
		if len(entries) == 0 {
			t.Fatal("Expected audit log entries before clear")
		}

		// Clear the log
		p.ClearAuditLog()

		// Verify entries cleared
		entries = p.GetAuditLog()
		if len(entries) != 0 {
			t.Errorf("Expected 0 entries after clear, got %d", len(entries))
		}
	})

	t.Run("ClearAuditLog allows continued processing", func(t *testing.T) {
		auditConfig := html.DefaultAuditConfig()
		auditConfig.Enabled = true

		config := html.DefaultConfig()
		config.Audit = auditConfig
		p, _ := html.New(config)
		defer p.Close()

		// First extraction
		p.Extract([]byte(`<html><body><script>1</script></body></html>`))
		p.ClearAuditLog()

		// Second extraction
		p.Extract([]byte(`<html><body><script>2</script></body></html>`))

		entries := p.GetAuditLog()
		if len(entries) == 0 {
			t.Error("Expected new audit entries after clear")
		}
	})
}
