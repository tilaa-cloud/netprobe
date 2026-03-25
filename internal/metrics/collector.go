package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusCollector implements prometheus.Collector for our metrics
type PrometheusCollector struct {
	storage         *MetricsStorage
	dimensionLabels []string // Original configured dimension labels from database
	customLabels    []string // Filtered custom labels (base labels removed)

	// Metric descriptors
	packetLoss prometheus.GaugeVec
	latencyMin prometheus.GaugeVec
	latencyMax prometheus.GaugeVec
	latencyAvg prometheus.GaugeVec
}

// filterBaseLabels removes base labels (destination_ip, method, responding_ip, responding_mac) from a list
// to prevent duplicates when building GaugeVec label names
func filterBaseLabels(labels []string) []string {
	baseLabels := map[string]bool{
		"destination_ip": true,
		"method":         true,
		"responding_ip":  true,
		"responding_mac": true,
	}

	filtered := make([]string, 0, len(labels))
	for _, label := range labels {
		if !baseLabels[label] {
			filtered = append(filtered, label)
		}
	}
	return filtered
}

// NewPrometheusCollector creates a new Prometheus collector with configurable dimensions
func NewPrometheusCollector(storage *MetricsStorage, dimensionLabels []string) *PrometheusCollector {
	// Filter out any base labels that might be in dimensionLabels (defensive)
	customLabels := filterBaseLabels(dimensionLabels)

	// Build label names: [destination_ip, method, responding_ip, responding_mac, ...custom dimensions...]
	labelNames := []string{"destination_ip", "method", "responding_ip", "responding_mac"}
	labelNames = append(labelNames, customLabels...)

	return &PrometheusCollector{
		storage:         storage,
		dimensionLabels: dimensionLabels, // Keep original for reference
		customLabels:    customLabels,    // Use filtered version for metrics
		packetLoss: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "netprobe_packet_loss_percent",
				Help: "Packet loss percentage for ping method",
			},
			labelNames,
		),
		latencyMin: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "netprobe_latency_min_ms",
				Help: "Minimum latency in milliseconds for ping method",
			},
			labelNames,
		),
		latencyMax: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "netprobe_latency_max_ms",
				Help: "Maximum latency in milliseconds for ping method",
			},
			labelNames,
		),
		latencyAvg: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "netprobe_latency_avg_ms",
				Help: "Average latency in milliseconds for ping method",
			},
			labelNames,
		),
	}
}

// Describe implements prometheus.Collector
func (c *PrometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	c.packetLoss.Describe(ch)
	c.latencyMin.Describe(ch)
	c.latencyMax.Describe(ch)
	c.latencyAvg.Describe(ch)
}

// Collect implements prometheus.Collector
func (c *PrometheusCollector) Collect(ch chan<- prometheus.Metric) {
	// Reset and update metrics from storage
	c.packetLoss.Reset()
	c.latencyMin.Reset()
	c.latencyMax.Reset()
	c.latencyAvg.Reset()

	for _, entry := range c.storage.GetAll() {
		key := entry.Key
		value := entry.Value

		// Build label values: [destination_ip, method, responding_ip, responding_mac, ...custom dimension values...]
		labels := []string{
			key.DestinationIP,
			key.Method,
			key.ResponsingIP,
			key.RespondingMac,
		}

		// Add custom dimension values in the order they were configured
		for _, dimLabel := range c.customLabels {
			if val, ok := key.Dimensions[dimLabel]; ok {
				labels = append(labels, val)
			} else {
				labels = append(labels, "") // Use empty string if dimension not found
			}
		}

		c.packetLoss.WithLabelValues(labels...).Set(value.PacketLossPercent)
		c.latencyMin.WithLabelValues(labels...).Set(value.LatencyMinMS)
		c.latencyMax.WithLabelValues(labels...).Set(value.LatencyMaxMS)
		c.latencyAvg.WithLabelValues(labels...).Set(value.LatencyAvgMS)
	}

	// Collect all metrics
	c.packetLoss.Collect(ch)
	c.latencyMin.Collect(ch)
	c.latencyMax.Collect(ch)
	c.latencyAvg.Collect(ch)
}
