package html

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/cybergodev/html/internal"
)

// AuditEventType represents the type of security audit event.
type AuditEventType string

const (
	// AuditEventBlockedTag indicates a dangerous HTML tag was blocked.
	AuditEventBlockedTag AuditEventType = "blocked_tag"
	// AuditEventBlockedAttr indicates a dangerous attribute was blocked.
	AuditEventBlockedAttr AuditEventType = "blocked_attr"
	// AuditEventBlockedURL indicates a dangerous URL was blocked.
	AuditEventBlockedURL AuditEventType = "blocked_url"
	// AuditEventInputViolation indicates an input validation failure.
	AuditEventInputViolation AuditEventType = "input_violation"
	// AuditEventDepthViolation indicates a depth limit violation.
	AuditEventDepthViolation AuditEventType = "depth_violation"
	// AuditEventTimeout indicates a processing timeout occurred.
	AuditEventTimeout AuditEventType = "timeout"
	// AuditEventEncodingIssue indicates an encoding detection issue.
	AuditEventEncodingIssue AuditEventType = "encoding_issue"
	// AuditEventPathTraversal indicates a path traversal attempt.
	AuditEventPathTraversal AuditEventType = "path_traversal"
)

// AuditLevel represents the severity level of an audit event.
type AuditLevel string

const (
	// AuditLevelInfo indicates informational events.
	AuditLevelInfo AuditLevel = "info"
	// AuditLevelWarning indicates warning events.
	AuditLevelWarning AuditLevel = "warning"
	// AuditLevelCritical indicates critical security events.
	AuditLevelCritical AuditLevel = "critical"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	EventType AuditEventType `json:"event_type"`
	Level     AuditLevel     `json:"level"`
	Message   string         `json:"message"`
	Tag       string         `json:"tag,omitempty"`
	Attribute string         `json:"attribute,omitempty"`
	URL       string         `json:"url,omitempty"`
	InputSize int            `json:"input_size,omitempty"`
	MaxSize   int            `json:"max_size,omitempty"`
	Depth     int            `json:"depth,omitempty"`
	MaxDepth  int            `json:"max_depth,omitempty"`
	Path      string         `json:"path,omitempty"`
	RawValue  string         `json:"raw_value,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// AuditSink defines the interface for audit log destinations.
// Implementations can write audit entries to various destinations
// such as files, databases, message queues, or external logging services.
type AuditSink interface {
	// Write writes an audit entry to the sink.
	// It should be non-blocking and thread-safe.
	Write(entry AuditEntry)
	// Close releases any resources used by the sink.
	Close() error
}

// AuditConfig configures the audit logging behavior.
type AuditConfig struct {
	// Enabled determines whether audit logging is active.
	Enabled bool `json:"enabled"`

	// LogBlockedTags logs when dangerous HTML tags are removed.
	LogBlockedTags bool `json:"log_blocked_tags"`

	// LogBlockedAttrs logs when dangerous attributes are removed.
	LogBlockedAttrs bool `json:"log_blocked_attrs"`

	// LogBlockedURLs logs when dangerous URLs are blocked.
	LogBlockedURLs bool `json:"log_blocked_urls"`

	// LogInputViolations logs input size and validation violations.
	LogInputViolations bool `json:"log_input_violations"`

	// LogDepthViolations logs depth limit violations.
	LogDepthViolations bool `json:"log_depth_violations"`

	// LogTimeouts logs processing timeout events.
	LogTimeouts bool `json:"log_timeouts"`

	// LogEncodingIssues logs encoding detection issues.
	LogEncodingIssues bool `json:"log_encoding_issues"`

	// LogPathTraversal logs path traversal attempts.
	LogPathTraversal bool `json:"log_path_traversal"`

	// Sink is the destination for audit entries.
	// If nil and audit is enabled, a default logger is used.
	Sink AuditSink `json:"-"`

	// IncludeRawValues includes raw attribute/URL values in logs.
	// Warning: This may log potentially malicious content.
	IncludeRawValues bool `json:"include_raw_values"`

	// MaxRawValueLength limits the length of raw values logged.
	MaxRawValueLength int `json:"max_raw_value_length"`
}

// DefaultAuditConfig returns the default audit configuration.
// By default, audit logging is disabled.
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		Enabled:            false,
		LogBlockedTags:     true,
		LogBlockedAttrs:    true,
		LogBlockedURLs:     true,
		LogInputViolations: true,
		LogDepthViolations: true,
		LogTimeouts:        true,
		LogEncodingIssues:  true,
		LogPathTraversal:   true,
		IncludeRawValues:   false,
		MaxRawValueLength:  200,
	}
}

// HighSecurityAuditConfig returns an audit configuration for high-security environments.
// All logging is enabled with raw value inclusion for forensic analysis.
func HighSecurityAuditConfig() AuditConfig {
	return AuditConfig{
		Enabled:            true,
		LogBlockedTags:     true,
		LogBlockedAttrs:    true,
		LogBlockedURLs:     true,
		LogInputViolations: true,
		LogDepthViolations: true,
		LogTimeouts:        true,
		LogEncodingIssues:  true,
		LogPathTraversal:   true,
		IncludeRawValues:   true,
		MaxRawValueLength:  500,
	}
}

// isEmpty checks if the AuditConfig has all zero values.
func (c AuditConfig) isEmpty() bool {
	return !c.Enabled &&
		!c.LogBlockedTags &&
		!c.LogBlockedAttrs &&
		!c.LogBlockedURLs &&
		!c.LogInputViolations &&
		!c.LogDepthViolations &&
		!c.LogTimeouts &&
		!c.LogEncodingIssues &&
		!c.LogPathTraversal &&
		!c.IncludeRawValues &&
		c.MaxRawValueLength == 0 &&
		c.Sink == nil
}

// AuditCollector collects audit entries during processing.
// It is designed to be thread-safe for concurrent use.
type AuditCollector struct {
	mu      sync.Mutex
	entries []AuditEntry
	config  AuditConfig
	sink    AuditSink
	wg      sync.WaitGroup // WaitGroup for async sink writes
}

// NewAuditCollector creates a new audit collector with the given configuration.
func NewAuditCollector(config AuditConfig) *AuditCollector {
	sink := config.Sink
	if sink == nil && config.Enabled {
		sink = NewLoggerAuditSink()
	}
	return &AuditCollector{
		entries: make([]AuditEntry, 0),
		config:  config,
		sink:    sink,
	}
}

// Record adds an audit entry to the collector.
func (c *AuditCollector) Record(entry AuditEntry) {
	if c == nil || !c.config.Enabled {
		return
	}

	entry.Timestamp = time.Now().UTC()

	// Truncate raw values if needed
	if c.config.MaxRawValueLength > 0 {
		if len(entry.RawValue) > c.config.MaxRawValueLength {
			entry.RawValue = entry.RawValue[:c.config.MaxRawValueLength] + "..."
		}
	}

	// Remove raw values if not configured to include them
	if !c.config.IncludeRawValues {
		entry.RawValue = ""
	}

	c.mu.Lock()
	c.entries = append(c.entries, entry)
	c.mu.Unlock()

	// Write to sink asynchronously with proper synchronization
	if c.sink != nil {
		c.wg.Add(1)
		go func(e AuditEntry) {
			defer c.wg.Done()
			c.sink.Write(e)
		}(entry)
	}
}

// Wait blocks until all pending async sink writes have completed.
// This is useful for ensuring all audit entries are written before
// test completion or processor shutdown.
func (c *AuditCollector) Wait() {
	if c == nil {
		return
	}
	c.wg.Wait()
}

// RecordBlockedTag records a blocked tag event.
func (c *AuditCollector) RecordBlockedTag(tag string) {
	if c == nil || !c.config.Enabled || !c.config.LogBlockedTags {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventBlockedTag,
		Level:     AuditLevelWarning,
		Message:   fmt.Sprintf("Blocked dangerous HTML tag: %s", tag),
		Tag:       tag,
	})
}

// RecordBlockedAttr records a blocked attribute event.
func (c *AuditCollector) RecordBlockedAttr(attr, value string) {
	if c == nil || !c.config.Enabled || !c.config.LogBlockedAttrs {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventBlockedAttr,
		Level:     AuditLevelWarning,
		Message:   fmt.Sprintf("Blocked dangerous attribute: %s", attr),
		Attribute: attr,
		RawValue:  value,
	})
}

// RecordBlockedURL records a blocked URL event.
func (c *AuditCollector) RecordBlockedURL(url, reason string) {
	if c == nil || !c.config.Enabled || !c.config.LogBlockedURLs {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventBlockedURL,
		Level:     AuditLevelWarning,
		Message:   fmt.Sprintf("Blocked dangerous URL: %s", reason),
		URL:       url,
		RawValue:  url,
	})
}

// RecordInputViolation records an input validation violation.
func (c *AuditCollector) RecordInputViolation(size, maxSize int, violationType string) {
	if c == nil || !c.config.Enabled || !c.config.LogInputViolations {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventInputViolation,
		Level:     AuditLevelCritical,
		Message:   fmt.Sprintf("Input validation violation: %s", violationType),
		InputSize: size,
		MaxSize:   maxSize,
	})
}

// RecordDepthViolation records a depth limit violation.
func (c *AuditCollector) RecordDepthViolation(depth, maxDepth int) {
	if c == nil || !c.config.Enabled || !c.config.LogDepthViolations {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventDepthViolation,
		Level:     AuditLevelWarning,
		Message:   fmt.Sprintf("Depth limit exceeded: %d > %d", depth, maxDepth),
		Depth:     depth,
		MaxDepth:  maxDepth,
	})
}

// RecordTimeout records a processing timeout event.
func (c *AuditCollector) RecordTimeout(timeout time.Duration) {
	if c == nil || !c.config.Enabled || !c.config.LogTimeouts {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventTimeout,
		Level:     AuditLevelWarning,
		Message:   fmt.Sprintf("Processing timeout exceeded: %v", timeout),
		Metadata:  map[string]any{"timeout": timeout.String()},
	})
}

// RecordEncodingIssue records an encoding detection issue.
func (c *AuditCollector) RecordEncodingIssue(encoding, message string) {
	if c == nil || !c.config.Enabled || !c.config.LogEncodingIssues {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventEncodingIssue,
		Level:     AuditLevelInfo,
		Message:   message,
		Metadata:  map[string]any{"encoding": encoding},
	})
}

// RecordPathTraversal records a path traversal attempt.
func (c *AuditCollector) RecordPathTraversal(path string) {
	if c == nil || !c.config.Enabled || !c.config.LogPathTraversal {
		return
	}
	c.Record(AuditEntry{
		EventType: AuditEventPathTraversal,
		Level:     AuditLevelCritical,
		Message:   "Path traversal attempt detected",
		Path:      path,
		RawValue:  path,
	})
}

// GetEntries returns all collected audit entries.
func (c *AuditCollector) GetEntries() []AuditEntry {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]AuditEntry, len(c.entries))
	copy(result, c.entries)
	return result
}

// Clear removes all collected entries.
func (c *AuditCollector) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make([]AuditEntry, 0)
}

// Close closes the audit collector and its sink.
// It waits for all pending async sink writes to complete before closing.
func (c *AuditCollector) Close() error {
	if c == nil {
		return nil
	}
	// Wait for all async sink writes to complete
	c.wg.Wait()
	if c.sink != nil {
		return c.sink.Close()
	}
	return nil
}

// ---- Built-in Audit Sinks ----

// LoggerAuditSink writes audit entries to a standard logger.
type LoggerAuditSink struct {
	logger *log.Logger
}

// NewLoggerAuditSink creates a new sink that writes to the default logger.
func NewLoggerAuditSink() *LoggerAuditSink {
	return &LoggerAuditSink{
		logger: log.New(os.Stderr, "[AUDIT] ", log.LstdFlags),
	}
}

// NewLoggerAuditSinkWithWriter creates a new sink that writes to the specified writer.
func NewLoggerAuditSinkWithWriter(w io.Writer) *LoggerAuditSink {
	return &LoggerAuditSink{
		logger: log.New(w, "[AUDIT] ", log.LstdFlags),
	}
}

// Write writes an audit entry to the logger.
func (s *LoggerAuditSink) Write(entry AuditEntry) {
	if s == nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		s.logger.Printf("Failed to marshal audit entry: %v", err)
		return
	}
	s.logger.Printf("%s", data)
}

// Close is a no-op for the logger sink.
func (s *LoggerAuditSink) Close() error {
	return nil
}

// ChannelAuditSink sends audit entries to a channel.
// Useful for integrating with external logging systems.
type ChannelAuditSink struct {
	ch     chan AuditEntry
	done   chan struct{}
	closed sync.Once
}

// NewChannelAuditSink creates a new sink that sends entries to a channel.
// The channel must be consumed by the caller to prevent blocking.
func NewChannelAuditSink(bufferSize int) *ChannelAuditSink {
	return &ChannelAuditSink{
		ch:   make(chan AuditEntry, bufferSize),
		done: make(chan struct{}),
	}
}

// Write sends an audit entry to the channel.
// If the channel is full, the entry is dropped to prevent blocking.
func (s *ChannelAuditSink) Write(entry AuditEntry) {
	if s == nil {
		return
	}
	select {
	case s.ch <- entry:
	default:
		// Channel full, drop the entry
	}
}

// Channel returns the channel for receiving audit entries.
func (s *ChannelAuditSink) Channel() <-chan AuditEntry {
	return s.ch
}

// Close closes the channel.
// Safe to call multiple times - only the first call will close the channel.
func (s *ChannelAuditSink) Close() error {
	if s == nil {
		return nil
	}
	s.closed.Do(func() {
		close(s.done)
		close(s.ch)
	})
	return nil
}

// WriterAuditSink writes audit entries to an io.Writer as JSON lines.
type WriterAuditSink struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewWriterAuditSink creates a new sink that writes to the specified writer.
func NewWriterAuditSink(w io.Writer) *WriterAuditSink {
	return &WriterAuditSink{writer: w}
}

// Write writes an audit entry to the writer.
func (s *WriterAuditSink) Write(entry AuditEntry) {
	if s == nil || s.writer == nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writer.Write(data)
	s.writer.Write([]byte("\n"))
}

// Close is a no-op for the writer sink.
func (s *WriterAuditSink) Close() error {
	return nil
}

// MultiSink wraps multiple sinks into a single sink.
type MultiSink struct {
	sinks []AuditSink
}

// NewMultiSink creates a new sink that writes to all provided sinks.
func NewMultiSink(sinks ...AuditSink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

// Write writes an audit entry to all sinks.
func (s *MultiSink) Write(entry AuditEntry) {
	if s == nil {
		return
	}
	for _, sink := range s.sinks {
		if sink != nil {
			sink.Write(entry)
		}
	}
}

// Close closes all sinks.
func (s *MultiSink) Close() error {
	if s == nil {
		return nil
	}
	var lastErr error
	for _, sink := range s.sinks {
		if sink != nil {
			if err := sink.Close(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// FilteredSink filters audit entries before writing to the underlying sink.
type FilteredSink struct {
	sink   AuditSink
	filter func(AuditEntry) bool
}

// NewFilteredSink creates a new sink that filters entries.
func NewFilteredSink(sink AuditSink, filter func(AuditEntry) bool) *FilteredSink {
	return &FilteredSink{sink: sink, filter: filter}
}

// Write writes an audit entry if it passes the filter.
func (s *FilteredSink) Write(entry AuditEntry) {
	if s == nil || s.sink == nil {
		return
	}
	if s.filter == nil || s.filter(entry) {
		s.sink.Write(entry)
	}
}

// Close closes the underlying sink.
func (s *FilteredSink) Close() error {
	if s == nil || s.sink == nil {
		return nil
	}
	return s.sink.Close()
}

// LevelFilteredSink only writes entries at or above the specified level.
type LevelFilteredSink struct {
	sink     AuditSink
	minLevel AuditLevel
}

// NewLevelFilteredSink creates a new sink that filters by level.
func NewLevelFilteredSink(sink AuditSink, minLevel AuditLevel) *LevelFilteredSink {
	return &LevelFilteredSink{sink: sink, minLevel: minLevel}
}

// Write writes an audit entry if it meets the minimum level.
func (s *LevelFilteredSink) Write(entry AuditEntry) {
	if s == nil || s.sink == nil {
		return
	}
	if s.meetsLevel(entry.Level) {
		s.sink.Write(entry)
	}
}

// Close closes the underlying sink.
func (s *LevelFilteredSink) Close() error {
	if s == nil || s.sink == nil {
		return nil
	}
	return s.sink.Close()
}

func (s *LevelFilteredSink) meetsLevel(level AuditLevel) bool {
	levels := map[AuditLevel]int{
		AuditLevelInfo:     0,
		AuditLevelWarning:  1,
		AuditLevelCritical: 2,
	}
	return levels[level] >= levels[s.minLevel]
}

// auditRecorderAdapter adapts AuditCollector to internal.AuditRecorder interface.
type auditRecorderAdapter struct {
	collector *AuditCollector
}

// RecordBlockedTag records a blocked tag event.
func (a *auditRecorderAdapter) RecordBlockedTag(tag string) {
	if a.collector != nil {
		a.collector.RecordBlockedTag(tag)
	}
}

// RecordBlockedAttr records a blocked attribute event.
func (a *auditRecorderAdapter) RecordBlockedAttr(attr, value string) {
	if a.collector != nil {
		a.collector.RecordBlockedAttr(attr, value)
	}
}

// RecordBlockedURL records a blocked URL event.
func (a *auditRecorderAdapter) RecordBlockedURL(url, reason string) {
	if a.collector != nil {
		a.collector.RecordBlockedURL(url, reason)
	}
}

// Ensure auditRecorderAdapter implements internal.AuditRecorder.
var _ internal.AuditRecorder = (*auditRecorderAdapter)(nil)
