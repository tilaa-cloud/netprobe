package metrics

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// MetricKey represents the labels for a Prometheus metric
// Uses a composite key approach to support flexible dimensions
type MetricKey struct {
	DestinationIP string            // Always required
	Method        string            // Always required (icmp, arp, or ndp)
	ResponsingIP  string            // IP that responded (may differ from destination in some scenarios)
	RespondingMac string            // MAC address from ARP/NDP response
	Dimensions    map[string]string // Custom dimensions from database
}

// String returns a stable string representation of the MetricKey for use as a map key
// This ensures consistent hashing despite map iteration order
// Note: RespondingMac is NOT included because it can change between failed and successful pings
// and shouldn't affect metric uniqueness (a latency metric is the same whether it failed or succeeded)
func (mk MetricKey) String() string {
	parts := []string{
		mk.DestinationIP,
		mk.Method,
		mk.ResponsingIP,
	}

	// Sort dimension keys for stable ordering
	if len(mk.Dimensions) > 0 {
		keys := make([]string, 0, len(mk.Dimensions))
		for k := range mk.Dimensions {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s:%s", k, mk.Dimensions[k]))
		}
	}

	return strings.Join(parts, "|")
}

// MetricValue represents the actual metric values
type MetricValue struct {
	PacketLossPercent float64
	LatencyMinMS      float64
	LatencyMaxMS      float64
	LatencyAvgMS      float64
	Timestamp         time.Time
}
