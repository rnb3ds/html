package html

import (
	"errors"
	"testing"
)

// TestUniformErrorBatch covers the unexported uniformErrorBatch helper directly.
// It is the only constructor path for an all-items-failed BatchResult, so the
// index-correspondence and counter invariants are worth pinning down at the unit
// level (batch_test.go is an external test package and cannot reach it).
func TestUniformErrorBatch(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("batch setup failed")

	tests := []struct {
		name string
		n    int
		err  error
	}{
		{"zero items", 0, sentinel},
		{"single item", 1, sentinel},
		{"three items", 3, sentinel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			br := uniformErrorBatch(tt.n, tt.err)
			if br == nil {
				t.Fatal("uniformErrorBatch returned nil")
			}
			if len(br.Results) != tt.n {
				t.Errorf("len(Results) = %d, want %d", len(br.Results), tt.n)
			}
			if len(br.Errors) != tt.n {
				t.Errorf("len(Errors) = %d, want %d", len(br.Errors), tt.n)
			}
			if br.Failed != tt.n {
				t.Errorf("Failed = %d, want %d", br.Failed, tt.n)
			}
			if br.Success != 0 {
				t.Errorf("Success = %d, want 0", br.Success)
			}
			if br.Cancelled != 0 {
				t.Errorf("Cancelled = %d, want 0", br.Cancelled)
			}
			for i, e := range br.Errors {
				if !errors.Is(e, tt.err) {
					t.Errorf("Errors[%d] = %v, want %v", i, e, tt.err)
				}
			}
		})
	}
}
