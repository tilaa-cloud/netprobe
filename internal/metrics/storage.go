package metrics

import "sync"

// MetricEntry pairs a MetricKey with its MetricValue
type MetricEntry struct {
	Key   MetricKey
	Value MetricValue
}

// MetricsStorage provides thread-safe storage for metrics
// Uses string keys based on MetricKey.String() for flexible dimension support
type MetricsStorage struct {
	mu         sync.RWMutex
	metrics    map[string]MetricValue
	metricKeys map[string]MetricKey // Store original MetricKey for label reconstruction
}

// NewMetricsStorage creates a new metrics storage instance
func NewMetricsStorage() *MetricsStorage {
	return &MetricsStorage{
		metrics:    make(map[string]MetricValue),
		metricKeys: make(map[string]MetricKey),
	}
}

// Update stores or updates a metric value
func (s *MetricsStorage) Update(key MetricKey, value MetricValue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyStr := key.String()
	s.metrics[keyStr] = value
	s.metricKeys[keyStr] = key
}

// GetAll returns a list of all stored metrics with their keys
// Returns a slice instead of a map since MetricKey contains a map field (not hashable)
func (s *MetricsStorage) GetAll() []MetricEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a slice with all entries
	result := make([]MetricEntry, 0, len(s.metrics))
	for keyStr, value := range s.metrics {
		key := s.metricKeys[keyStr]
		result = append(result, MetricEntry{
			Key:   key,
			Value: value,
		})
	}
	return result
}

// Get retrieves a specific metric by key
func (s *MetricsStorage) Get(key MetricKey) (MetricValue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyStr := key.String()
	val, ok := s.metrics[keyStr]
	return val, ok
}

// Clear removes all metrics
func (s *MetricsStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = make(map[string]MetricValue)
	s.metricKeys = make(map[string]MetricKey)
}
