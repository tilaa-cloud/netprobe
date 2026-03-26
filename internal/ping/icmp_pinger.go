package ping

import (
	"context"
	"net"
	"time"

	//nolint:staticcheck // Package is deprecated but maintained by community and still functional
	goping "github.com/go-ping/ping"

	"netprobe/internal/logger"
	"netprobe/internal/target"
)

// ICMPPinger implements the Pinger interface for ICMP
type ICMPPinger struct {
	timeoutMS int
	count     int
	selector  *InterfaceSelector
}

// NewICMPPinger creates a new ICMP pinger
func NewICMPPinger(timeoutMS, count int) *ICMPPinger {
	return &ICMPPinger{
		timeoutMS: timeoutMS,
		count:     count,
		selector:  NewInterfaceSelector(),
	}
}

// Ping executes an ICMP ping using the go-ping library
func (p *ICMPPinger) Ping(ctx context.Context, t target.Target, method string) (PingResult, error) {
	result := PingResult{
		Target:        t,
		Method:        method,
		ResponsingIP:  t.DestinationIP,
		RespondingMac: "", // ICMP doesn't provide MAC addresses
		PacketsSent:   p.count,
		Timestamp:     time.Now(),
	}

	// Parse the destination IP
	destIP := net.ParseIP(t.DestinationIP)
	if destIP == nil {
		logger.Debug("[ICMP] Invalid IP address: %s", t.DestinationIP)
		result.Success = false
		result.PacketsLost = p.count
		result.PacketLossPercent = 100.0
		return result, nil
	}

	// Try to bind to the correct source interface for IPv6
	// For IPv4, let kernel routing handle it (binding to gateway IP breaks ping)
	var sourceIP string
	if destIP.To4() == nil {
		// IPv6 target - try to find IPv6 interface
		iface, err := p.selector.FindInterfaceForIPv6(destIP)
		if err == nil && iface != nil {
			// Get the IP address from this interface
			addrs, err := iface.Addrs()
			if err == nil && len(addrs) > 0 {
				for _, addr := range addrs {
					if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() == nil {
						sourceIP = ipNet.IP.String()
						logger.Debug("[ICMP] Using source IP %s on interface %s for %s", sourceIP, iface.Name, t.DestinationIP)
						break
					}
				}
			}
		}
	}

	// Create pinger
	pinger, err := goping.NewPinger(t.DestinationIP)
	if err != nil {
		logger.Debug("[ICMP] Failed to create pinger for %s: %v", t.DestinationIP, err)
		result.Success = false
		result.PacketsLost = p.count
		result.PacketLossPercent = 100.0
		return result, err
	}

	// Enable privileged mode for raw socket ICMP
	pinger.SetPrivileged(true)

	// Bind to the source address if we found one
	if sourceIP != "" {
		pinger.Source = sourceIP
	}

	// Set ping parameters
	pinger.Count = p.count
	pinger.Timeout = time.Duration(p.timeoutMS) * time.Millisecond

	// Run the ping
	err = pinger.Run()
	if err != nil {
		logger.Debug("[ICMP] Ping failed for %s: %v", t.DestinationIP, err)
		result.Success = false
		result.PacketsLost = p.count
		result.PacketLossPercent = 100.0
		return result, err
	}

	// Get statistics
	stats := pinger.Statistics()
	result.Success = stats.PacketsRecv > 0
	result.PacketsLost = stats.PacketsSent - stats.PacketsRecv
	result.PacketLossPercent = stats.PacketLoss
	result.LatencyMinMS = float64(stats.MinRtt.Milliseconds())
	result.LatencyMaxMS = float64(stats.MaxRtt.Milliseconds())
	result.LatencyAvgMS = float64(stats.AvgRtt.Milliseconds())

	if result.Success {
		logger.Debug("[ICMP] Success: %s responded (loss=%.1f%%, latency=%.2fms)",
			t.DestinationIP, result.PacketLossPercent, result.LatencyAvgMS)
	} else {
		logger.Debug("[ICMP] Failed: %s no response (loss=100%%)", t.DestinationIP)
	}

	return result, nil
}
