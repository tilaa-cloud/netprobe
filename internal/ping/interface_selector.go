package ping

import (
	"fmt"
	"net"

	"netprobe/internal/logger"
)

// InterfaceSelector handles finding the correct network interface for ping methods.
// Both ARP (IPv4) and NDP (IPv6) need to select an interface based on subnet membership.
type InterfaceSelector struct {
	// Cache interfaces to avoid repeated system calls
	interfaces []net.Interface
	// IPv6SubnetBits defines the subnet prefix length for IPv6 matching (default: 64)
	// Common values: 64 (most networks), 56 (some ISPs), 48 (regional allocation)
	IPv6SubnetBits int
}

// NewInterfaceSelector creates a new interface selector with default IPv6 /64 subnet matching
func NewInterfaceSelector() *InterfaceSelector {
	return &InterfaceSelector{
		IPv6SubnetBits: 64, // Default to /64 subnet matching for IPv6
	}
}

// NewInterfaceSelectorWithIPv6Bits creates a new interface selector with custom IPv6 subnet bits
// subnetBits: number of bits to match for IPv6 addresses (e.g., 64 for /64, 56 for /56)
func NewInterfaceSelectorWithIPv6Bits(subnetBits int) *InterfaceSelector {
	if subnetBits < 1 || subnetBits > 128 {
		logger.Debug("[InterfaceSelector] Invalid IPv6 subnet bits %d, using default 64", subnetBits)
		subnetBits = 64
	}
	return &InterfaceSelector{
		IPv6SubnetBits: subnetBits,
	}
}

// FindInterfaceForIPv4 finds the network interface that can reach the target IPv4 address.
// Uses CIDR subnet checking to respect configured network masks (/8, /16, /24, /32, etc).
func (s *InterfaceSelector) FindInterfaceForIPv4(targetIP net.IP) (*net.Interface, error) {
	if targetIP == nil {
		return nil, fmt.Errorf("target IP is nil")
	}

	// Ensure it's actually IPv4
	if targetIP.To4() == nil {
		return nil, fmt.Errorf("not an IPv4 address: %s", targetIP)
	}

	interfaces, err := s.getInterfaces()
	if err != nil {
		return nil, err
	}

	// First pass: find interface with exact subnet match using CIDR
	for i := range interfaces {
		iface := &interfaces[i]
		if !s.isInterfaceUsable(iface) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// Check if target IP is on the same subnet as any address on this interface
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.Contains(targetIP) {
					logger.Debug("[InterfaceSelector] Found IPv4 interface for %s: %s (subnet: %s)", targetIP, iface.Name, ipNet)
					return iface, nil
				}
			}
		}
	}

	// Fallback: return first usable non-loopback, up interface
	for i := range interfaces {
		iface := &interfaces[i]
		if s.isInterfaceUsable(iface) {
			logger.Debug("[InterfaceSelector] No exact subnet match for %s, using fallback: %s", targetIP, iface.Name)
			return iface, nil
		}
	}

	return nil, fmt.Errorf("no suitable network interface found for IPv4 target %s", targetIP)
}

// FindInterfaceForIPv6 finds the network interface that can reach the target IPv6 address.
// Uses configurable subnet prefix matching (default /64, but can be customized for /48, /56, etc).
func (s *InterfaceSelector) FindInterfaceForIPv6(targetIP net.IP) (*net.Interface, error) {
	if targetIP == nil {
		return nil, fmt.Errorf("target IP is nil")
	}

	// Ensure it's actually IPv6
	if targetIP.To4() != nil {
		return nil, fmt.Errorf("not an IPv6 address: %s", targetIP)
	}

	interfaces, err := s.getInterfaces()
	if err != nil {
		return nil, err
	}

	// Find interface with matching /64 subnet
	for i := range interfaces {
		iface := &interfaces[i]
		if !s.isInterfaceUsable(iface) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				ip := ipNet.IP
				// Only check IPv6 addresses
				if ip.To4() == nil && ip.To16() != nil {
					if s.isIPv6OnSameSubnet(ip, targetIP) {
						logger.Debug("[InterfaceSelector] Found IPv6 interface for %s: %s (subnet: %s, bits: %d)", targetIP, iface.Name, ipNet, s.IPv6SubnetBits)
						return iface, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no suitable network interface found for IPv6 target %s", targetIP)
}

// IsInterfaceUsable checks if an interface is up and not loopback
// This is a public method for testing and external use
func (s *InterfaceSelector) IsInterfaceUsable(iface *net.Interface) bool {
	if iface == nil {
		return false
	}
	// Must be UP and not loopback
	return iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0
}

// isInterfaceUsable checks if an interface is up and not loopback (internal use)
func (s *InterfaceSelector) isInterfaceUsable(iface *net.Interface) bool {
	return s.IsInterfaceUsable(iface)
}

// isIPv6OnSameSubnet checks if two IPv6 addresses are on the same subnet
// Uses the configured IPv6SubnetBits for the comparison
func (s *InterfaceSelector) isIPv6OnSameSubnet(ip1, ip2 net.IP) bool {
	return s.IsIPv6OnSameSubnet(ip1, ip2)
}

// IsIPv6OnSameSubnet checks if two IPv6 addresses are on the same subnet
// Uses the configured IPv6SubnetBits for the comparison (e.g., 64 for /64, 56 for /56)
// This is a public method for testing and external use
func (s *InterfaceSelector) IsIPv6OnSameSubnet(ip1, ip2 net.IP) bool {
	// Ensure both are IPv6 addresses
	if ip1.To4() != nil || ip2.To4() != nil {
		return false
	}

	if len(ip1) < 16 || len(ip2) < 16 {
		return false
	}

	// Calculate how many bytes and bits to compare
	bytesToCompare := s.IPv6SubnetBits / 8
	remainingBits := s.IPv6SubnetBits % 8

	// Compare full bytes
	for i := 0; i < bytesToCompare; i++ {
		if ip1[i] != ip2[i] {
			return false
		}
	}

	// Compare remaining bits if any
	if remainingBits > 0 {
		// Create a mask for the remaining bits
		mask := byte(0xFF << uint(8-remainingBits))
		if (ip1[bytesToCompare] & mask) != (ip2[bytesToCompare] & mask) {
			return false
		}
	}

	return true
}

// getInterfaces lazily loads and caches the interface list
func (s *InterfaceSelector) getInterfaces() ([]net.Interface, error) {
	if s.interfaces == nil {
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil, fmt.Errorf("failed to list network interfaces: %w", err)
		}
		s.interfaces = interfaces
	}
	return s.interfaces, nil
}

// ClearCache clears the interface cache (useful for testing)
func (s *InterfaceSelector) ClearCache() {
	s.interfaces = nil
}
