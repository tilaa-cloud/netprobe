package ping

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"netprobe/internal/logger"
	"netprobe/internal/target"
)

// NDPPinger implements real Neighbor Discovery Protocol for IPv6 targets
// Sends Neighbor Solicitation (NS) and listens for Neighbor Advertisement (NA)
type NDPPinger struct {
	timeoutMS int
	selector  *InterfaceSelector
}

// NewNDPPinger creates a new NDP pinger
func NewNDPPinger(timeoutMS int) *NDPPinger {
	return &NDPPinger{
		timeoutMS: timeoutMS,
		selector:  NewInterfaceSelector(),
	}
}

// Ping performs real NDP by sending Neighbor Solicitation messages
// Returns success if target responds with Neighbor Advertisement
func (p *NDPPinger) Ping(ctx context.Context, t target.Target, method string) (PingResult, error) {
	start := time.Now()
	targetIP := t.DestinationIP

	result := PingResult{
		Target:        t,
		Method:        method,
		ResponsingIP:  targetIP,
		RespondingMac: "", // Will be populated from NDP response
		PacketsSent:   1,
		Timestamp:     time.Now(),
	}

	// Check context before starting
	select {
	case <-ctx.Done():
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, ctx.Err()
	default:
	}

	// Parse the target as an IPv6 address
	ip := net.ParseIP(targetIP)
	if ip == nil {
		logger.Debug("[NDP] Invalid IP address: %s", targetIP)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, fmt.Errorf("invalid IP address: %s", targetIP)
	}

	// Ensure it's IPv6
	if ip.To4() != nil {
		logger.Debug("[NDP] Not an IPv6 address: %s", targetIP)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, fmt.Errorf("NDP requires IPv6 address, got: %s", targetIP)
	}

	// Find the network interface to use for the target IPv6
	iface, err := p.selector.FindInterfaceForIPv6(ip)
	if err != nil {
		logger.Debug("[NDP] Could not find IPv6 interface: %v", err)
		result.Success = false
		result.PacketsLost = 1
		result.PacketLossPercent = 100.0
		return result, err
	}

	// Try real NDP: send Neighbor Solicitation and wait for Neighbor Advertisement
	success, mac, latency := p.neighborSolicitation(ip, iface)

	if success {
		elapsed := time.Since(start).Milliseconds()
		result.Success = true
		result.PacketsLost = 0
		result.PacketLossPercent = 0.0
		result.ResponsingIP = targetIP
		result.RespondingMac = mac
		result.LatencyMinMS = float64(latency)
		result.LatencyMaxMS = float64(latency)
		result.LatencyAvgMS = float64(latency)
		logger.Debug("[NDP] Success: %s responded with MAC %s (latency=%dms)", targetIP, mac, elapsed)
		return result, nil
	}

	// NDP failed
	logger.Debug("[NDP] Failed: %s is unreachable", targetIP)
	result.Success = false
	result.PacketsLost = 1
	result.PacketLossPercent = 100.0
	result.LatencyMinMS = 0
	result.LatencyMaxMS = 0
	result.LatencyAvgMS = 0
	return result, nil
}

// neighborSolicitation sends an ICMPv6 Neighbor Solicitation and listens for NA
// Returns (success bool, MAC address string, latency in milliseconds)
func (p *NDPPinger) neighborSolicitation(targetIP net.IP, iface *net.Interface) (bool, string, int64) {
	start := time.Now()

	// Verify this interface can actually reach the target
	srcIP, err := getInterfaceIPv6(iface)
	if err != nil {
		logger.Debug("[NDP] Could not get IPv6 address for %s: %v", iface.Name, err)
		return false, "", -1
	}

	logger.Debug("[NDP] Using interface %s with IPv6 %s to reach %s", iface.Name, srcIP, targetIP)

	// Check if target is on the same subnet as interface (should be true since FindInterfaceForIPv6 already checked this)
	if !p.selector.isIPv6OnSameSubnet(srcIP, targetIP) {
		// Fallback: try to find the correct interface (shouldn't normally reach here)
		iface, err = p.selector.FindInterfaceForIPv6(targetIP)
		if err != nil {
			logger.Debug("[NDP] Could not find interface for %s: %v", targetIP, err)
			return false, "", -1
		}

		srcIP, err = getInterfaceIPv6(iface)
		if err != nil {
			logger.Debug("[NDP] Could not get IPv6 address for %s: %v", iface.Name, err)
			return false, "", -1
		}
	}

	// Open pcap device for packet capture
	handle, err := pcap.OpenLive(iface.Name, 65535, true, time.Millisecond*10)
	if err != nil {
		logger.Debug("[NDP] Failed to open pcap on %s: %v", iface.Name, err)
		return false, "", -1
	}
	defer handle.Close()

	// Set filter to capture ICMPv6 packets
	err = handle.SetBPFFilter("icmp6")
	if err != nil {
		logger.Debug("[NDP] Warning: BPF filter failed: %v", err)
	}

	// Channel to communicate response from listener goroutine
	type naResponse struct {
		mac     string
		latency int64
	}
	responseChan := make(chan naResponse, 1)
	errorChan := make(chan error, 1)
	stopChan := make(chan struct{})

	// Start listener goroutine BEFORE sending
	go func() {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		packetSource.NoCopy = true
		for {
			select {
			case <-stopChan:
				return
			case packet := <-packetSource.Packets():
				if packet == nil {
					return
				}

				// Parse IPv6 layer
				ipv6Layer := packet.Layer(layers.LayerTypeIPv6)
				if ipv6Layer == nil {
					continue
				}
				ipv6Packet := ipv6Layer.(*layers.IPv6)

				// Must come from target IP
				if !ipv6Packet.SrcIP.Equal(targetIP) {
					continue
				}

				// Parse ICMPv6 layer
				icmpv6Layer := packet.Layer(layers.LayerTypeICMPv6)
				if icmpv6Layer == nil {
					continue
				}
				icmpv6Packet := icmpv6Layer.(*layers.ICMPv6)

				// Must be Neighbor Advertisement (type 136)
				if icmpv6Packet.TypeCode.Type() != layers.ICMPv6TypeNeighborAdvertisement {
					continue
				}

				// Extract MAC from Ethernet header
				ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
				if ethernetLayer == nil {
					continue
				}

				ethernet := ethernetLayer.(*layers.Ethernet)
				latency := time.Since(start).Milliseconds()
				mac := ethernet.SrcMAC.String()
				logger.Debug("[NDP] Success: %s responded with MAC %s (latency=%dms)", targetIP, mac, latency)
				responseChan <- naResponse{mac: mac, latency: latency}
				return
			}
		}
	}()

	// Small delay to ensure listener is ready
	time.Sleep(10 * time.Millisecond)

	// Build and send Neighbor Solicitation packet
	eth := layers.Ethernet{
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       net.HardwareAddr{0x33, 0x33, 0xff, targetIP[13], targetIP[14], targetIP[15]}, // IPv6 multicast MAC
		EthernetType: layers.EthernetTypeIPv6,
	}

	ipv6 := layers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		Length:       32,
		NextHeader:   layers.IPProtocolICMPv6,
		HopLimit:     255,
		SrcIP:        srcIP,
		DstIP:        solicitedNodeMulticast(targetIP),
	}

	icmpv6 := layers.ICMPv6{
		TypeCode: layers.CreateICMPv6TypeCode(layers.ICMPv6TypeNeighborSolicitation, 0),
	}

	ns := layers.ICMPv6NeighborSolicitation{
		TargetAddress: targetIP,
	}

	icmpv6.SetNetworkLayerForChecksum(&ipv6)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	err = gopacket.SerializeLayers(buffer, opts, &eth, &ipv6, &icmpv6, &ns)
	if err != nil {
		logger.Debug("[NDP] Failed to serialize NS packet for %s: %v", targetIP, err)
		close(stopChan)
		return false, "", -1
	}

	// Send the packet
	err = handle.WritePacketData(buffer.Bytes())
	if err != nil {
		logger.Debug("[NDP] Failed to send NS to %s: %v", targetIP, err)
		close(stopChan)
		return false, "", -1
	}

	logger.Debug("[NDP] Sent NS to %s from %s on %s", targetIP, srcIP, iface.Name)

	// Wait for response with timeout
	select {
	case resp := <-responseChan:
		close(stopChan)
		return true, resp.mac, resp.latency
	case err := <-errorChan:
		logger.Debug("[NDP] Error waiting for NA: %v", err)
		close(stopChan)
		return false, "", -1
	case <-time.After(time.Duration(p.timeoutMS) * time.Millisecond):
		logger.Debug("[NDP] Timeout (%.1fs) waiting for NA from %s", float64(p.timeoutMS)/1000.0, targetIP)
		close(stopChan)
		return false, "", -1
	}
}

// solicitedNodeMulticast returns the IPv6 solicited node multicast address
func solicitedNodeMulticast(ip net.IP) net.IP {
	// ff02::1:ff + last 3 octets of target address
	result := net.IP{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01, 0xff, ip[13], ip[14], ip[15]}
	return result
}

// getInterfaceIPv6 gets the IPv6 address of an interface
// Prefers link-local addresses (fe80::/10) for NDP operations
func getInterfaceIPv6(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	var globalAddr net.IP
	var linkLocalAddr net.IP

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.To4() == nil && ipNet.IP.To16() != nil {
				ip := ipNet.IP
				// Check if it's a link-local address (fe80::/10)
				if ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
					linkLocalAddr = ip
				} else {
					// Store first global address as fallback
					if globalAddr == nil {
						globalAddr = ip
					}
				}
			}
		}
	}

	// Prefer link-local address for NDP
	if linkLocalAddr != nil {
		return linkLocalAddr, nil
	}
	if globalAddr != nil {
		return globalAddr, nil
	}

	return nil, fmt.Errorf("no IPv6 address found on interface %s", iface.Name)
}
