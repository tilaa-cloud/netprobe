package ping

import (
	"time"

	"netprobe/internal/target"
)

// PingResult represents the outcome of a single ping operation
type PingResult struct {
	Target            target.Target
	Method            string // "icmp", "arp", or "ndp"
	ResponsingIP      string // IP address that responded (typically same as destination)
	RespondingMac     string // MAC address (from ARP/NDP response)
	Success           bool
	PacketsSent       int
	PacketsLost       int
	PacketLossPercent float64
	LatencyMinMS      float64
	LatencyMaxMS      float64
	LatencyAvgMS      float64
	Timestamp         time.Time
}
