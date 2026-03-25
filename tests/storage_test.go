package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"netprobe/internal/metrics"
)

// TestIntegration_MetricsStorage tests that metrics can be stored and retrieved
func TestIntegration_MetricsStorage(t *testing.T) {
	store := metrics.NewMetricsStorage()

	// Create a test metric
	key := metrics.MetricKey{
		DestinationIP: "10.0.0.1",
		Method:        "icmp",
		ResponsingIP:  "10.0.0.1",
		RespondingMac: "",
		Dimensions: map[string]string{
			"customer_id": "customer1",
			"vlan":        "prod",
			"pod":         "us-west-2",
			"host":        "server-01",
		},
	}

	value := metrics.MetricValue{
		PacketLossPercent: 0.0,
		LatencyMinMS:      1.5,
		LatencyMaxMS:      2.5,
		LatencyAvgMS:      2.0,
		Timestamp:         time.Now(),
	}

	// Store metric
	store.Update(key, value)

	// Retrieve and verify
	all := store.GetAll()
	assert.Len(t, all, 1)
	assert.Equal(t, value.LatencyAvgMS, all[0].Value.LatencyAvgMS)
}
