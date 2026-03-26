package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"syscall"
	"time"

	"netprobe/internal/config"
	"netprobe/internal/logger"
	"netprobe/internal/ping"
	"netprobe/internal/target"
)

// checkRawSocketPermission verifies that we have CAP_NET_RAW capability
// needed for ARP, NDP, and raw ICMP operations
func checkRawSocketPermission() bool {
	// Try to create a raw socket (requires CAP_NET_RAW)
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, syscall.ETH_P_ALL)
	if err != nil {
		return false
	}
	_ = syscall.Close(fd) // nolint: errcheck
	return true
}

func main() {
	// Initialize logger from environment
	logger.InitFromEnv()

	// Check for required capabilities
	if !checkRawSocketPermission() {
		logger.Fatal("[INIT] CAP_NET_RAW capability not available - required for ARP and NDP operations. To fix: run 'setcap cap_net_raw=ep /path/to/netprobe-ping' or use sudo")
	}

	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Get remaining args (method and target)
	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: netprobe-ping [options] <method> <target>\n")
		fmt.Fprintf(os.Stderr, "Methods: icmp, arp, ndp\n")
		fmt.Fprintf(os.Stderr, "Example: netprobe-ping icmp 8.8.8.8\n")
		fmt.Fprintf(os.Stderr, "         netprobe-ping arp 192.168.1.1\n")
		fmt.Fprintf(os.Stderr, "         netprobe-ping ndp fd00::1\n")
		os.Exit(1)
	}

	method := args[0]
	targetIP := args[1]

	// Validate method
	if method != "icmp" && method != "arp" && method != "ndp" {
		logger.Error("Invalid method: %s (must be icmp, arp, or ndp)", method)
		os.Exit(1)
	}

	// Use default timeouts if config doesn't exist
	icmpTimeoutMS := 5000
	icmpCount := 1
	arpTimeoutMS := 5000
	ndpTimeoutMS := 5000

	// Try to load config for overrides, but don't fail if it doesn't exist
	if cfg, err := config.LoadConfig(*configPath); err == nil {
		icmpTimeoutMS = cfg.ICMP.TimeoutMS
		icmpCount = cfg.ICMP.Count
		arpTimeoutMS = cfg.ARP.TimeoutMS
		ndpTimeoutMS = cfg.NDP.TimeoutMS
	}

	// Create pinger with configured timeouts
	pinger := ping.NewCompositePinger(
		icmpTimeoutMS,
		icmpCount,
		arpTimeoutMS,
		ndpTimeoutMS,
	)

	// Create a test target
	testTarget := target.Target{
		DestinationIP: targetIP,
		Dimensions:    make(map[string]string),
	}

	// Execute the ping with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := pinger.Ping(ctx, testTarget, method)
	if err != nil {
		logger.Error("Ping error: %v", err)
		os.Exit(1)
	}

	// Display result
	fmt.Printf("Method: %s\n", result.Method)
	fmt.Printf("Target: %s\n", result.Target.DestinationIP)
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Packets Sent: %d\n", result.PacketsSent)
	fmt.Printf("Packets Lost: %d\n", result.PacketsLost)
	fmt.Printf("Packet Loss: %.1f%%\n", result.PacketLossPercent)
	if result.Success {
		fmt.Printf("Latency - Min: %.2fms, Max: %.2fms, Avg: %.2fms\n",
			result.LatencyMinMS, result.LatencyMaxMS, result.LatencyAvgMS)
		if result.RespondingMac != "" {
			fmt.Printf("Responding MAC: %s\n", result.RespondingMac)
		}
	}

	if result.Success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
