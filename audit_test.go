package html

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestAuditLoggingEnabled tests that audit logging captures security events
func TestAuditLoggingEnabled(t *testing.T) {
	t.Parallel()

	auditConfig := DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.IncludeRawValues = true

	cfg := DefaultConfig()
	cfg.Audit = auditConfig

	p, err := New(cfg)
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
		case AuditEventBlockedTag:
			foundBlockedTag = true
			if entry.Tag != "script" {
				t.Errorf("Expected blocked tag 'script', got '%s'", entry.Tag)
			}
		case AuditEventBlockedAttr:
			foundBlockedAttr = true
			if entry.Attribute != "onclick" {
				t.Errorf("Expected blocked attribute 'onclick', got '%s'", entry.Attribute)
			}
		case AuditEventBlockedURL:
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

	auditConfig := DefaultAuditConfig()
	auditConfig.Enabled = false

	cfg := DefaultConfig()
	cfg.Audit = auditConfig

	p, err := New(cfg)
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

	auditConfig := DefaultAuditConfig()
	auditConfig.Enabled = true

	cfg := DefaultConfig()
	cfg.MaxInputSize = 1000 // 1KB limit
	cfg.Audit = auditConfig

	p, err := New(cfg)
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
		if entry.EventType == AuditEventInputViolation {
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

	auditConfig := DefaultAuditConfig()
	auditConfig.Enabled = true

	cfg := DefaultConfig()
	cfg.Audit = auditConfig

	p, err := New(cfg)
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
		if entry.EventType == AuditEventPathTraversal {
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

	config := HighSecurityAuditConfig()
	collector := newAuditCollector(config)
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

	auditConfig := DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.IncludeRawValues = true
	auditConfig.MaxRawValueLength = 50

	cfg := DefaultConfig()
	cfg.Audit = auditConfig

	p, err := New(cfg)
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
		if entry.EventType == AuditEventBlockedAttr {
			if len(entry.RawValue) > 53 { // 50 + "..."
				t.Errorf("Raw value should be truncated, got length %d", len(entry.RawValue))
			}
		}
	}
}

// TestAuditExcludeRawValues tests that raw values can be excluded
func TestAuditExcludeRawValues(t *testing.T) {
	t.Parallel()

	auditConfig := DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.IncludeRawValues = false

	cfg := DefaultConfig()
	cfg.Audit = auditConfig

	p, err := New(cfg)
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
		if entry.EventType == AuditEventBlockedAttr {
			if entry.RawValue != "" {
				t.Errorf("Raw value should be empty when excluded, got '%s'", entry.RawValue)
			}
		}
	}
}

// TestAuditSinks tests all audit sink implementations
// Consolidates Logger, Writer, and Channel sink tests into a single table-driven test
func TestAuditSinks(t *testing.T) {
	t.Parallel()

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: AuditEventBlockedTag,
		Level:     AuditLevelWarning,
		Message:   "Test message",
		Tag:       "script",
	}

	tests := []struct {
		name       string
		create     func() (AuditSink, func() string)
		validate   func(t *testing.T, output string)
		needsClose bool
	}{
		{
			name: "LoggerSink",
			create: func() (AuditSink, func() string) {
				var buf bytes.Buffer
				sink := NewLoggerAuditSinkWithWriter(&buf)
				return sink, buf.String
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "Test message") {
					t.Errorf("Expected output to contain message, got: %s", output)
				}
			},
			needsClose: true,
		},
		{
			name: "WriterSink",
			create: func() (AuditSink, func() string) {
				var buf bytes.Buffer
				sink := NewWriterAuditSink(&buf)
				return sink, buf.String
			},
			validate: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Expected output from writer sink")
				}
				var decoded AuditEntry
				if err := json.Unmarshal([]byte(output), &decoded); err != nil {
					t.Errorf("Output should be valid JSON: %v", err)
				}
			},
			needsClose: false,
		},
		{
			name: "ChannelSink",
			create: func() (AuditSink, func() string) {
				sink := NewChannelAuditSink(10)
				result := make(chan string, 1)
				go func() {
					select {
					case received := <-sink.Channel():
						result <- received.Message
					case <-time.After(time.Second):
						result <- "timeout"
					}
				}()
				return sink, func() string {
					select {
					case r := <-result:
						return r
					case <-time.After(100 * time.Millisecond):
						return ""
					}
				}
			},
			validate: func(t *testing.T, output string) {
				if output != entry.Message {
					t.Errorf("Expected message '%s', got '%s'", entry.Message, output)
				}
			},
			needsClose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink, getOutput := tt.create()
			if tt.needsClose {
				defer sink.Close()
			}

			sink.Write(entry)
			tt.validate(t, getOutput())
		})
	}
}

// TestMultiSink tests the multi sink
func TestMultiSink(t *testing.T) {
	t.Parallel()

	var buf1, buf2 bytes.Buffer
	sink1 := NewWriterAuditSink(&buf1)
	sink2 := NewWriterAuditSink(&buf2)

	multi := NewMultiSink(sink1, sink2)
	defer multi.Close()

	entry := AuditEntry{
		Timestamp: time.Now(),
		EventType: AuditEventBlockedTag,
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
	writerSink := NewWriterAuditSink(&buf)
	filtered := NewLevelFilteredSink(writerSink, AuditLevelWarning)
	defer filtered.Close()

	// Info level should be filtered out
	infoEntry := AuditEntry{
		Timestamp: time.Now(),
		EventType: AuditEventEncodingIssue,
		Level:     AuditLevelInfo,
		Message:   "Info message",
	}
	filtered.Write(infoEntry)

	if buf.String() != "" {
		t.Error("Info level should be filtered out")
	}

	// Warning level should pass
	warningEntry := AuditEntry{
		Timestamp: time.Now(),
		EventType: AuditEventBlockedTag,
		Level:     AuditLevelWarning,
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
		writerSink := NewWriterAuditSink(&buf)

		// Filter that only allows blocked tag events
		filter := func(entry AuditEntry) bool {
			return entry.EventType == AuditEventBlockedTag
		}

		filtered := NewFilteredSink(writerSink, filter)
		defer filtered.Close()

		// Entry that should pass filter
		blockedTagEntry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventBlockedTag,
			Level:     AuditLevelWarning,
			Message:   "Blocked tag",
		}
		filtered.Write(blockedTagEntry)

		if buf.String() == "" {
			t.Error("BlockedTag event should pass filter")
		}

		buf.Reset()

		// Entry that should be filtered out
		encodingEntry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventEncodingIssue,
			Level:     AuditLevelInfo,
			Message:   "Encoding issue",
		}
		filtered.Write(encodingEntry)

		if buf.String() != "" {
			t.Error("EncodingIssue event should be filtered out")
		}
	})

	t.Run("filter allows all when filter is nil", func(t *testing.T) {
		var buf bytes.Buffer
		writerSink := NewWriterAuditSink(&buf)

		filtered := NewFilteredSink(writerSink, nil)
		defer filtered.Close()

		entry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventBlockedTag,
			Message:   "Test message",
		}
		filtered.Write(entry)

		if buf.String() == "" {
			t.Error("Nil filter should allow all entries")
		}
	})

	t.Run("filter by level", func(t *testing.T) {
		var buf bytes.Buffer
		writerSink := NewWriterAuditSink(&buf)

		// Filter that only allows critical level
		filter := func(entry AuditEntry) bool {
			return entry.Level == AuditLevelCritical
		}

		filtered := NewFilteredSink(writerSink, filter)
		defer filtered.Close()

		// Warning should be filtered out
		warningEntry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventBlockedTag,
			Level:     AuditLevelWarning,
			Message:   "Warning",
		}
		filtered.Write(warningEntry)

		if buf.String() != "" {
			t.Error("Warning level should be filtered out")
		}

		// Critical should pass
		criticalEntry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventInputViolation,
			Level:     AuditLevelCritical,
			Message:   "Critical",
		}
		filtered.Write(criticalEntry)

		if buf.String() == "" {
			t.Error("Critical level should pass filter")
		}
	})

	t.Run("close propagates to underlying sink", func(t *testing.T) {
		var buf bytes.Buffer
		writerSink := NewWriterAuditSink(&buf)
		filtered := NewFilteredSink(writerSink, nil)

		// Close should not return error
		if err := filtered.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	t.Run("nil sink handles gracefully", func(t *testing.T) {
		filtered := NewFilteredSink(nil, func(entry AuditEntry) bool {
			return true
		})

		// Write should not panic with nil sink
		entry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventBlockedTag,
			Message:   "Test",
		}
		filtered.Write(entry)

		// Close should handle nil sink
		if err := filtered.Close(); err != nil {
			t.Errorf("Close() with nil sink returned error: %v", err)
		}
	})

	t.Run("nil filtered sink handles gracefully", func(t *testing.T) {
		var filtered *FilteredSink

		// Write should not panic with nil filtered sink
		entry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventBlockedTag,
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

	cfg := HighSecurityConfig()

	if !cfg.Audit.Enabled {
		t.Error("HighSecurityConfig should have audit enabled")
	}
	if !cfg.Audit.LogBlockedTags {
		t.Error("HighSecurityConfig should log blocked tags")
	}
	if !cfg.Audit.LogBlockedAttrs {
		t.Error("HighSecurityConfig should log blocked attributes")
	}
	if !cfg.Audit.LogBlockedURLs {
		t.Error("HighSecurityConfig should log blocked URLs")
	}
	if !cfg.Audit.IncludeRawValues {
		t.Error("HighSecurityConfig should include raw values for forensics")
	}
}

// TestAuditEntryJSON tests JSON serialization of audit entries
func TestAuditEntryJSON(t *testing.T) {
	t.Parallel()

	entry := AuditEntry{
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		EventType: AuditEventBlockedTag,
		Level:     AuditLevelWarning,
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

	var decoded AuditEntry
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
	customSink := NewWriterAuditSink(&buf)

	auditConfig := DefaultAuditConfig()
	auditConfig.Enabled = true
	auditConfig.Sink = customSink

	cfg := DefaultConfig()
	cfg.Audit = auditConfig

	p, err := New(cfg)
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
		cfg := DefaultConfig()
		cfg.Audit.Enabled = false
		p, _ := New(cfg)
		defer p.Close()

		htmlContent := []byte(`<html><body><h1>Test</h1><p>Content</p></body></html>`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			p.Extract(htmlContent)
		}
	})

	b.Run("enabled", func(b *testing.B) {
		cfg := DefaultConfig()
		cfg.Audit.Enabled = true
		p, _ := New(cfg)
		defer p.Close()

		htmlContent := []byte(`<html><body><h1>Test</h1><p>Content</p></body></html>`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			p.Extract(htmlContent)
			p.ClearAuditLog()
		}
	})

	b.Run("enabled_with_xss", func(b *testing.B) {
		cfg := DefaultConfig()
		cfg.Audit.Enabled = true
		p, _ := New(cfg)
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
		sink := NewChannelAuditSink(10)

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
		sink := NewWriterAuditSink(&buf)

		config := HighSecurityAuditConfig()
		config.Sink = sink
		collector := newAuditCollector(config)
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
		var collector *auditCollector
		// Should not panic
		collector.Wait()
	})

	t.Run("Wait with disabled audit is safe", func(t *testing.T) {
		config := DefaultAuditConfig() // Disabled by default
		collector := newAuditCollector(config)
		defer collector.Close()

		collector.RecordBlockedTag("script")
		collector.Wait() // Should complete immediately
	})

	t.Run("Wait allows multiple calls", func(t *testing.T) {
		config := HighSecurityAuditConfig()
		collector := newAuditCollector(config)
		defer collector.Close()

		collector.RecordBlockedTag("script")
		collector.Wait()
		collector.Wait() // Second call should be safe
		collector.Wait() // Third call should be safe
	})
}

// TestAuditRecordMethods exercises every Record* path of auditCollector in one
// table, replacing the previously triplicated "records event" / "log-flag
// disabled" / "disabled audit" / "nil collector" subtests that were spread
// across TestRecordEncodingIssue, TestRecordBlockedAttrDisabled,
// TestRecordBlockedURLDisabled, TestRecordDepthViolation and TestRecordTimeout.
// It also covers RecordBlockedTag, the one Record* method that had no dedicated
// test. Each row supplies the method call, the config flag gating it, the
// expected event type, and any method-specific entry assertions.
func TestAuditRecordMethods(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		record         func(c *auditCollector)
		makeConfig     func() AuditConfig // Enabled with this method's log flag on
		disableLogFlag func(c *AuditConfig)
		wantEventType  AuditEventType
		checkEntry     func(t *testing.T, e AuditEntry)
	}{
		{
			name:   "blocked tag",
			record: func(c *auditCollector) { c.RecordBlockedTag("script") },
			makeConfig: func() AuditConfig {
				cfg := DefaultAuditConfig()
				cfg.Enabled = true
				cfg.LogBlockedTags = true
				return cfg
			},
			disableLogFlag: func(c *AuditConfig) { c.LogBlockedTags = false },
			wantEventType:  AuditEventBlockedTag,
			checkEntry: func(t *testing.T, e AuditEntry) {
				t.Helper()
				if e.Level != AuditLevelWarning {
					t.Errorf("Level = %s, want %s", e.Level, AuditLevelWarning)
				}
				if e.Tag != "script" {
					t.Errorf("Tag = %q, want %q", e.Tag, "script")
				}
			},
		},
		{
			name:   "blocked attr",
			record: func(c *auditCollector) { c.RecordBlockedAttr("onclick", "alert(1)") },
			makeConfig: func() AuditConfig {
				cfg := DefaultAuditConfig()
				cfg.Enabled = true
				cfg.LogBlockedAttrs = true
				cfg.IncludeRawValues = true
				return cfg
			},
			disableLogFlag: func(c *AuditConfig) { c.LogBlockedAttrs = false },
			wantEventType:  AuditEventBlockedAttr,
			checkEntry: func(t *testing.T, e AuditEntry) {
				t.Helper()
				if e.Level != AuditLevelWarning {
					t.Errorf("Level = %s, want %s", e.Level, AuditLevelWarning)
				}
				if e.Attribute != "onclick" {
					t.Errorf("Attribute = %q, want %q", e.Attribute, "onclick")
				}
				if e.RawValue != "alert(1)" {
					t.Errorf("RawValue = %q, want %q", e.RawValue, "alert(1)")
				}
			},
		},
		{
			name:   "blocked url",
			record: func(c *auditCollector) { c.RecordBlockedURL("javascript:alert(1)", "javascript scheme") },
			makeConfig: func() AuditConfig {
				cfg := DefaultAuditConfig()
				cfg.Enabled = true
				cfg.LogBlockedURLs = true
				return cfg
			},
			disableLogFlag: func(c *AuditConfig) { c.LogBlockedURLs = false },
			wantEventType:  AuditEventBlockedURL,
			checkEntry: func(t *testing.T, e AuditEntry) {
				t.Helper()
				if e.Level != AuditLevelWarning {
					t.Errorf("Level = %s, want %s", e.Level, AuditLevelWarning)
				}
				if e.URL != "javascript:alert(1)" {
					t.Errorf("URL = %q, want %q", e.URL, "javascript:alert(1)")
				}
			},
		},
		{
			name:   "encoding issue",
			record: func(c *auditCollector) { c.RecordEncodingIssue("windows-1252", "invalid byte sequence") },
			makeConfig: func() AuditConfig {
				cfg := DefaultAuditConfig()
				cfg.Enabled = true
				cfg.LogEncodingIssues = true
				return cfg
			},
			disableLogFlag: func(c *AuditConfig) { c.LogEncodingIssues = false },
			wantEventType:  AuditEventEncodingIssue,
			checkEntry: func(t *testing.T, e AuditEntry) {
				t.Helper()
				if e.Level != AuditLevelInfo {
					t.Errorf("Level = %s, want %s", e.Level, AuditLevelInfo)
				}
				if e.Message != "invalid byte sequence" {
					t.Errorf("Message = %q, want %q", e.Message, "invalid byte sequence")
				}
				if e.Metadata["encoding"] != "windows-1252" {
					t.Errorf("Metadata[encoding] = %v, want %q", e.Metadata["encoding"], "windows-1252")
				}
			},
		},
		{
			name:   "depth violation",
			record: func(c *auditCollector) { c.RecordDepthViolation(150, 100) },
			makeConfig: func() AuditConfig {
				cfg := DefaultAuditConfig()
				cfg.Enabled = true
				cfg.LogDepthViolations = true
				return cfg
			},
			disableLogFlag: func(c *AuditConfig) { c.LogDepthViolations = false },
			wantEventType:  AuditEventDepthViolation,
			checkEntry: func(t *testing.T, e AuditEntry) {
				t.Helper()
				if e.Depth != 150 {
					t.Errorf("Depth = %d, want 150", e.Depth)
				}
				if e.MaxDepth != 100 {
					t.Errorf("MaxDepth = %d, want 100", e.MaxDepth)
				}
			},
		},
		{
			name:   "timeout",
			record: func(c *auditCollector) { c.RecordTimeout(5 * time.Second) },
			makeConfig: func() AuditConfig {
				cfg := DefaultAuditConfig()
				cfg.Enabled = true
				cfg.LogTimeouts = true
				return cfg
			},
			disableLogFlag: func(c *AuditConfig) { c.LogTimeouts = false },
			wantEventType:  AuditEventTimeout,
			checkEntry: func(t *testing.T, e AuditEntry) {
				t.Helper()
				if e.Level != AuditLevelWarning {
					t.Errorf("Level = %s, want %s", e.Level, AuditLevelWarning)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			t.Run("records event", func(t *testing.T) {
				t.Parallel()
				collector := newAuditCollector(tc.makeConfig())
				defer collector.Close()
				tc.record(collector)

				entries := collector.GetEntries()
				if len(entries) != 1 {
					t.Fatalf("Expected 1 entry, got %d", len(entries))
				}
				if entries[0].EventType != tc.wantEventType {
					t.Errorf("EventType = %s, want %s", entries[0].EventType, tc.wantEventType)
				}
				if tc.checkEntry != nil {
					tc.checkEntry(t, entries[0])
				}
			})

			t.Run("log flag disabled does not record", func(t *testing.T) {
				t.Parallel()
				cfg := tc.makeConfig()
				tc.disableLogFlag(&cfg)
				collector := newAuditCollector(cfg)
				defer collector.Close()
				tc.record(collector)

				if entries := collector.GetEntries(); len(entries) != 0 {
					t.Errorf("Expected 0 entries with log flag disabled, got %d", len(entries))
				}
			})

			t.Run("disabled audit does not record", func(t *testing.T) {
				t.Parallel()
				cfg := tc.makeConfig()
				cfg.Enabled = false
				collector := newAuditCollector(cfg)
				defer collector.Close()
				tc.record(collector)

				if entries := collector.GetEntries(); len(entries) != 0 {
					t.Errorf("Expected 0 entries with audit disabled, got %d", len(entries))
				}
			})

			t.Run("nil collector does not panic", func(t *testing.T) {
				t.Parallel()
				var collector *auditCollector
				tc.record(collector) // must not panic
			})
		})
	}
}

// TestRecordEncodingIssueRecordsMultiple preserves the aggregation check that
// distinct encoding issues each produce their own entry (previously a subtest of
// TestRecordEncodingIssue).
func TestRecordEncodingIssueRecordsMultiple(t *testing.T) {
	t.Parallel()

	config := DefaultAuditConfig()
	config.Enabled = true
	config.LogEncodingIssues = true
	collector := newAuditCollector(config)
	defer collector.Close()

	encodings := []string{"utf-8", "windows-1252", "iso-8859-1", "shift_jis", "gbk"}
	for _, enc := range encodings {
		collector.RecordEncodingIssue(enc, "detection failed")
	}

	if entries := collector.GetEntries(); len(entries) != len(encodings) {
		t.Errorf("Expected %d entries, got %d", len(encodings), len(entries))
	}
}

// TestClearAuditLog tests the ClearAuditLog method on Processor
func TestClearAuditLog(t *testing.T) {
	t.Parallel()

	t.Run("ClearAuditLog clears processor audit entries", func(t *testing.T) {
		auditConfig := DefaultAuditConfig()
		auditConfig.Enabled = true

		cfg := DefaultConfig()
		cfg.Audit = auditConfig
		p, err := New(cfg)
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
		auditConfig := DefaultAuditConfig()
		auditConfig.Enabled = true

		cfg := DefaultConfig()
		cfg.Audit = auditConfig
		p, _ := New(cfg)
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

// TestMultiSinkClose tests MultiSink Close method edge cases.
func TestMultiSinkClose(t *testing.T) {
	t.Parallel()

	t.Run("close with nil sinks", func(t *testing.T) {
		multi := NewMultiSink(nil, nil)
		err := multi.Close()
		if err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	t.Run("close with mixed nil and valid sinks", func(t *testing.T) {
		var buf bytes.Buffer
		sink := NewWriterAuditSink(&buf)
		multi := NewMultiSink(nil, sink, nil)

		err := multi.Close()
		if err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})
}

// TestLevelFilteredSinkClose tests LevelFilteredSink Close method edge cases.
func TestLevelFilteredSinkClose(t *testing.T) {
	t.Parallel()

	t.Run("close with nil sink", func(t *testing.T) {
		filtered := NewLevelFilteredSink(nil, AuditLevelWarning)
		err := filtered.Close()
		if err != nil {
			t.Errorf("Close() with nil sink returned error: %v", err)
		}
	})
}

// TestLoggerAuditSinkWriteEdgeCases tests LoggerAuditSink Write edge cases.
func TestLoggerAuditSinkWriteEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("write with empty message", func(t *testing.T) {
		var buf bytes.Buffer
		sink := NewLoggerAuditSinkWithWriter(&buf)

		entry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventBlockedTag,
			Message:   "",
		}

		sink.Write(entry)
		// Should still log something even with empty message
		if buf.String() == "" {
			t.Error("Expected some output even with empty message")
		}
	})
}

// TestWriterAuditSinkWriteEdgeCases tests WriterAuditSink Write edge cases.
func TestWriterAuditSinkWriteEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("write with complex metadata", func(t *testing.T) {
		var buf bytes.Buffer
		sink := NewWriterAuditSink(&buf)

		entry := AuditEntry{
			Timestamp: time.Now(),
			EventType: AuditEventBlockedTag,
			Message:   "Test",
			Metadata: map[string]any{
				"string": "value",
				"number": 123,
				"bool":   true,
				"nested": map[string]any{"key": "value"},
			},
		}

		sink.Write(entry)

		// Verify JSON is valid
		var decoded AuditEntry
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Errorf("Output should be valid JSON: %v", err)
		}
	})
}

// TestChannelAuditSinkDroppedCount covers ChannelAuditSink.DroppedCount
// (previously 0%) and the Write drop branch that feeds it: when the channel
// buffer is full, Write must drop the entry and increment the counter rather
// than block.
func TestChannelAuditSinkDroppedCount(t *testing.T) {
	t.Parallel()

	t.Run("nil sink is safe", func(t *testing.T) {
		var sink *ChannelAuditSink
		if got := sink.DroppedCount(); got != 0 {
			t.Errorf("nil DroppedCount = %d, want 0", got)
		}
		// Must not panic on a nil receiver.
		sink.Write(AuditEntry{})
	})

	t.Run("drops and counts when buffer full", func(t *testing.T) {
		// A buffer of size 0 is unbuffered, so every Write without a concurrent
		// reader takes the drop branch.
		sink := NewChannelAuditSink(0)
		defer sink.Close()

		for i := 0; i < 5; i++ {
			sink.Write(AuditEntry{EventType: AuditEventBlockedTag})
		}

		if got := sink.DroppedCount(); got != 5 {
			t.Errorf("DroppedCount = %d, want 5", got)
		}
	})

	t.Run("buffered writes are not counted as dropped", func(t *testing.T) {
		sink := NewChannelAuditSink(2)
		defer sink.Close()

		// Two writes fit in the buffer and must not be dropped.
		sink.Write(AuditEntry{})
		sink.Write(AuditEntry{})

		if got := sink.DroppedCount(); got != 0 {
			t.Errorf("DroppedCount = %d, want 0 for buffered writes", got)
		}

		// Drain so Close does not race with pending sends.
		<-sink.Channel()
		<-sink.Channel()
	})

	t.Run("drops resume after the buffer drains", func(t *testing.T) {
		sink := NewChannelAuditSink(1)
		defer sink.Close()

		sink.Write(AuditEntry{}) // fills the 1-slot buffer
		// Next write has nowhere to go -> dropped.
		sink.Write(AuditEntry{})
		if got := sink.DroppedCount(); got != 1 {
			t.Fatalf("DroppedCount = %d, want 1", got)
		}

		<-sink.Channel()         // free the slot
		sink.Write(AuditEntry{}) // now fits -> not dropped
		if got := sink.DroppedCount(); got != 1 {
			t.Errorf("DroppedCount = %d, want 1 (counter must not increase for a delivered write)", got)
		}
	})
}
