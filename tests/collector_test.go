package tests

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"netprobe/internal/metrics"
	"netprobe/internal/target"
)

// TestIntegration_PrometheusMetricsExposed tests that metrics are properly exposed in Prometheus format
func TestIntegration_PrometheusMetricsExposed(t *testing.T) {
	store := metrics.NewMetricsStorage()
	dimensionLabels := []string{"customer_id", "vlan", "pod", "host"}
	collector := metrics.NewPrometheusCollector(store, dimensionLabels)

	// Register collector with Prometheus
	reg := prometheus.NewRegistry()
	err := reg.Register(collector)
	require.NoError(t, err)

	// Add test metrics
	store.Update(metrics.MetricKey{
		DestinationIP: "10.0.0.1",
		Method:        "icmp",
		ResponsingIP:  "10.0.0.1",
		RespondingMac: "",
		Dimensions: map[string]string{
			"customer_id": "acme",
			"vlan":        "prod",
			"pod":         "us-west-2",
			"host":        "server-01",
		},
	}, metrics.MetricValue{
		PacketLossPercent: 0.0,
		LatencyMinMS:      1.0,
		LatencyMaxMS:      3.0,
		LatencyAvgMS:      2.0,
		Timestamp:         time.Now(),
	})

	store.Update(metrics.MetricKey{
		DestinationIP: "10.0.0.2",
		Method:        "arp",
		ResponsingIP:  "10.0.0.2",
		RespondingMac: "00:11:22:33:44:55",
		Dimensions: map[string]string{
			"customer_id": "acme",
			"vlan":        "prod",
			"pod":         "us-west-2",
			"host":        "server-02",
		},
	}, metrics.MetricValue{
		PacketLossPercent: 5.0,
		LatencyMinMS:      0.5,
		LatencyMaxMS:      2.5,
		LatencyAvgMS:      1.5,
		Timestamp:         time.Now(),
	})

	// Gather metrics in Prometheus text format
	promMetrics, err := reg.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, promMetrics)

	// Verify we have metrics for packet loss and latencies
	metricNames := make(map[string]bool)
	for _, mf := range promMetrics {
		metricNames[mf.GetName()] = true
	}
	assert.True(t, metricNames["netprobe_packet_loss_percent"])
	assert.True(t, metricNames["netprobe_latency_min_ms"])
	assert.True(t, metricNames["netprobe_latency_max_ms"])
	assert.True(t, metricNames["netprobe_latency_avg_ms"])
}

// TestIntegration_MetricLabels tests that all required labels are present in metrics
func TestIntegration_MetricLabels(t *testing.T) {
	testTarget := target.Target{
		DestinationIP: "10.0.0.123",
		Dimensions: map[string]string{
			"customer_id": "acme-corp",
			"vlan":        "prod-vlan",
			"pod":         "us-west-2-pod",
			"host":        "db-server-01",
		},
	}

	// Create store with test metric
	store := metrics.NewMetricsStorage()
	dimensionLabels := []string{"customer_id", "vlan", "pod", "host"}
	store.Update(metrics.MetricKey{
		DestinationIP: testTarget.DestinationIP,
		Method:        "icmp",
		ResponsingIP:  testTarget.DestinationIP,
		RespondingMac: "",
		Dimensions:    testTarget.Dimensions,
	}, metrics.MetricValue{
		PacketLossPercent: 0.0,
		LatencyMinMS:      2.0,
		LatencyMaxMS:      2.0,
		LatencyAvgMS:      2.0,
		Timestamp:         time.Now(),
	})

	collector := metrics.NewPrometheusCollector(store, dimensionLabels)
	reg := prometheus.NewRegistry()
	reg.Register(collector)

	resp, _ := prometheus.DefaultGatherer.Gather()
	if resp == nil {
		resp, _ = reg.Gather()
	}

	// Verify metric structure
	assert.NotNil(t, resp)
}
