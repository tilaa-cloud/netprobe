package target

import "context"

// TargetSource defines the interface for fetching targets from any source
type TargetSource interface {
	Fetch(ctx context.Context) ([]Target, error)
}

// EmptyTargetSource returns an empty list of targets
// Used when no database is configured
type EmptyTargetSource struct{}

// Fetch returns an empty target list
func (e *EmptyTargetSource) Fetch(ctx context.Context) ([]Target, error) {
	return []Target{}, nil
}

// NewEmptyTargetSource creates a new empty target source
func NewEmptyTargetSource() *EmptyTargetSource {
	return &EmptyTargetSource{}
}
