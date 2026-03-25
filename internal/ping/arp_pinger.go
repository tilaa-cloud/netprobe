package ping

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/j-keck/arping"

	"netprobe/internal/logger"
	"netprobe/internal/target"
)

// ARPPinger implements the Pinger interface for ARP
type ARPPinger struct {
	timeoutMS int
	selector  *InterfaceSelector
}

// NewARPPinger creates a new ARP pinger
func NewARPPinger(timeoutMS int) *ARPPinger {
	return &ARPPinger{
		timeoutMS: timeoutMS,
		selector:  NewInterfaceSelector(),
	}
}

// Ping executes an ARP ping and captures the responding MAC address
func (p *ARPPinger) Ping(ctx context.Context, t target.Target, method string) (PingResult, error) {
	start := time.Now()
	result := PingResult{
		Target:        t,
		Method:        method,
		ResponsingIP:  t.DestinationIP,
		RespondingMac: "", // Will be populated from ARP response
		PacketsSent:   1,
		Timestamp:     time.Now(),
	}

	// Parse the target IP
	ip := net.ParseIP(t.DestinationIP)
	if ip == nil {
		logger.Debug("[ARP] Invalid IP address: %s", t.DestinationIP)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, fmt.Errorf("invalid IP address: %s", t.DestinationIP)
	}

	// Create a channel to handle timeout
	type arpResult struct {
		hwAddr net.HardwareAddr
		err    error
	}
	resultChan := make(chan arpResult, 1)

	// Get the appropriate network interface for this destination
	iface, err := p.selector.FindInterfaceForIPv4(ip)
	if err != nil {
		logger.Debug("[ARP] Could not determine network interface for %s: %v", t.DestinationIP, err)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, nil
	}

	// Run ARP ping in a goroutine with timeout handling
	go func() {
		hwAddr, _, err := arping.PingOverIface(ip, *iface)
		resultChan <- arpResult{hwAddr, err}
	}()

	// Wait for result or timeout
	var arpErr error
	var hwAddr net.HardwareAddr
	select {
	case res := <-resultChan:
		hwAddr = res.hwAddr
		arpErr = res.err
	case <-ctx.Done():
		logger.Debug("[ARP] Timeout waiting for response from %s", t.DestinationIP)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, ctx.Err()
	case <-time.After(time.Duration(p.timeoutMS) * time.Millisecond):
		logger.Debug("[ARP] Timeout waiting for response from %s", t.DestinationIP)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, fmt.Errorf("ARP timeout")
	}

	if arpErr != nil {
		// ARP ping failed
		logger.Debug("[ARP] Failed to reach %s: %v", t.DestinationIP, arpErr)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		result.LatencyMinMS = 0
		result.LatencyMaxMS = 0
		result.LatencyAvgMS = 0
		return result, nil
	}

	// Success - target responded to ARP
	latency := time.Since(start).Milliseconds()
	result.Success = true
	result.PacketsLost = 0
	result.PacketLossPercent = 0.0
	result.ResponsingIP = t.DestinationIP
	result.RespondingMac = hwAddr.String() // Capture actual MAC address from ARP response
	result.LatencyMinMS = float64(latency)
	result.LatencyMaxMS = float64(latency)
	result.LatencyAvgMS = float64(latency)

	logger.Debug("[ARP] Success: %s responded with MAC %s (latency=%dms)",
		t.DestinationIP, result.RespondingMac, latency)

	return result, nil
}
