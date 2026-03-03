package internal

// AuditRecorder defines the interface for recording security audit events.
// This interface is used internally to decouple the sanitization code from
// the main audit implementation.
type AuditRecorder interface {
	// RecordBlockedTag records when a dangerous tag is removed.
	RecordBlockedTag(tag string)
	// RecordBlockedAttr records when a dangerous attribute is removed.
	RecordBlockedAttr(attr, value string)
	// RecordBlockedURL records when a dangerous URL is blocked.
	RecordBlockedURL(url, reason string)
}

// NoOpAuditRecorder is an audit recorder that does nothing.
// Used when audit logging is disabled.
type NoOpAuditRecorder struct{}

// RecordBlockedTag does nothing.
func (NoOpAuditRecorder) RecordBlockedTag(tag string) {}

// RecordBlockedAttr does nothing.
func (NoOpAuditRecorder) RecordBlockedAttr(attr, value string) {}

// RecordBlockedURL does nothing.
func (NoOpAuditRecorder) RecordBlockedURL(url, reason string) {}

// Ensure NoOpAuditRecorder implements AuditRecorder.
var _ AuditRecorder = NoOpAuditRecorder{}
