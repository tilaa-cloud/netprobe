package tests

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpserver "netprobe/internal/http"
	"netprobe/internal/metrics"
)

// TestIntegration_HTTPServerMetricsEndpoint tests that the HTTP server exposes /metrics endpoint
func TestIntegration_HTTPServerMetricsEndpoint(t *testing.T) {
	store := metrics.NewMetricsStorage()
	dimensionLabels := []string{"customer_id", "vlan", "pod", "host"}
	collector := metrics.NewPrometheusCollector(store, dimensionLabels)

	// Add test metric
	store.Update(metrics.MetricKey{
		DestinationIP: "10.0.0.1",
		Method:        "icmp",
		ResponsingIP:  "10.0.0.1",
		RespondingMac: "",
		Dimensions: map[string]string{
			"customer_id": "test-customer",
			"vlan":        "test-vlan",
			"pod":         "test-pod",
			"host":        "test-host",
		},
	}, metrics.MetricValue{
		PacketLossPercent: 2.5,
		LatencyMinMS:      1.0,
		LatencyMaxMS:      5.0,
		LatencyAvgMS:      3.0,
		Timestamp:         time.Now(),
	})

	// Start HTTP server
	server := httpserver.NewServer(":9099", collector) // Use non-standard port for testing
	go server.Start()
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Query /metrics endpoint
	resp, err := http.Get("http://localhost:9099/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Verify response contains expected metrics
	metricsText := string(body)
	assert.Contains(t, metricsText, "netprobe_packet_loss_percent")
	assert.Contains(t, metricsText, "netprobe_latency_avg_ms")
	assert.Contains(t, metricsText, "destination_ip=\"10.0.0.1\"")
	assert.Contains(t, metricsText, "method=\"icmp\"")
	assert.Contains(t, metricsText, "responding_ip=\"10.0.0.1\"")
	assert.Contains(t, metricsText, "responding_mac=\"\"")
	assert.Contains(t, metricsText, "customer_id=\"test-customer\"")
	assert.Contains(t, metricsText, "vlan=\"test-vlan\"")
	assert.Contains(t, metricsText, "pod=\"test-pod\"")
	assert.Contains(t, metricsText, "host=\"test-host\"")
}
