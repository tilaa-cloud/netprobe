package tests

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"netprobe/internal/ping"
)

// TestUnit_InterfaceSelector_IPv4_SubnetDetection tests IPv4 CIDR subnet detection
func TestUnit_InterfaceSelector_IPv4_SubnetDetection(t *testing.T) {
	testCases := []struct {
		name        string
		interfaceIP string
		maskBits    int
		targetIP    string
		shouldMatch bool
	}{
		{
			name:        "Basic /24 subnet match",
			interfaceIP: "192.168.1.1",
			maskBits:    24,
			targetIP:    "192.168.1.100",
			shouldMatch: true,
		},
		{
			name:        "Basic /24 subnet no match",
			interfaceIP: "192.168.1.1",
			maskBits:    24,
			targetIP:    "192.168.2.1",
			shouldMatch: false,
		},
		{
			name:        "/32 exact match only",
			interfaceIP: "10.0.0.1",
			maskBits:    32,
			targetIP:    "10.0.0.1",
			shouldMatch: true,
		},
		{
			name:        "/32 non-match",
			interfaceIP: "10.0.0.1",
			maskBits:    32,
			targetIP:    "10.0.0.2",
			shouldMatch: false,
		},
		{
			name:        "/16 subnet match",
			interfaceIP: "172.16.0.1",
			maskBits:    16,
			targetIP:    "172.16.255.255",
			shouldMatch: true,
		},
		{
			name:        "/8 subnet match",
			interfaceIP: "10.1.1.1",
			maskBits:    8,
			targetIP:    "10.255.255.255",
			shouldMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create an IPNet with the given interface IP and mask bits
			ip := net.ParseIP(tc.interfaceIP)
			require.NotNil(t, ip, "Failed to parse interface IP: %s", tc.interfaceIP)

			ipMask := net.CIDRMask(tc.maskBits, 32)
			ipNet := &net.IPNet{
				IP:   ip.Mask(ipMask),
				Mask: ipMask,
			}

			targetIP := net.ParseIP(tc.targetIP)
			require.NotNil(t, targetIP, "Failed to parse target IP: %s", tc.targetIP)

			// Test the CIDR Contains logic
			matches := ipNet.Contains(targetIP)
			assert.Equal(t, tc.shouldMatch, matches,
				"Expected subnet %s/%d %s target %s",
				ipNet.IP, tc.maskBits, map[bool]string{true: "to contain", false: "to not contain"}[tc.shouldMatch], tc.targetIP)
		})
	}
}

// TestUnit_InterfaceSelector_IPv6_SubnetDetection tests IPv6 /64 subnet detection
func TestUnit_InterfaceSelector_IPv6_SubnetDetection(t *testing.T) {
	selector := ping.NewInterfaceSelector()

	testCases := []struct {
		name       string
		ip1        string
		ip2        string
		sameSubnet bool
	}{
		{
			name:       "Same /64 subnet - typical case",
			ip1:        "2001:db8:85a3::8a2e:370:7334",
			ip2:        "2001:db8:85a3::1",
			sameSubnet: true,
		},
		{
			name:       "Same /64 subnet - minimal difference in host part",
			ip1:        "2001:db8:abcd:0000:0000:0000:0000:0000",
			ip2:        "2001:db8:abcd:0000:ffff:ffff:ffff:ffff",
			sameSubnet: true,
		},
		{
			name:       "Different /64 subnet - different byte 8",
			ip1:        "2001:db8:85a3::0001",
			ip2:        "2001:db8:85a3:0001::0001",
			sameSubnet: false,
		},
		{
			name:       "Different /64 subnet - different second segment",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db8:85a4::1",
			sameSubnet: false,
		},
		{
			name:       "Different /64 subnet - different first segment",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db9:85a3::1",
			sameSubnet: false,
		},
		{
			name:       "Link-local addresses - same /64",
			ip1:        "fe80::1",
			ip2:        "fe80::2",
			sameSubnet: true,
		},
		{
			name:       "Link-local vs global - different /64",
			ip1:        "fe80::1",
			ip2:        "2001:db8::1",
			sameSubnet: false,
		},
		{
			name:       "Loopback addresses",
			ip1:        "::1",
			ip2:        "::1",
			sameSubnet: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip1 := net.ParseIP(tc.ip1)
			ip2 := net.ParseIP(tc.ip2)
			require.NotNil(t, ip1, "Failed to parse ip1: %s", tc.ip1)
			require.NotNil(t, ip2, "Failed to parse ip2: %s", tc.ip2)

			sameSubnet := selector.IsIPv6OnSameSubnet(ip1, ip2)
			assert.Equal(t, tc.sameSubnet, sameSubnet,
				"Expected %s and %s to be on same /64 subnet: %v, got: %v",
				tc.ip1, tc.ip2, tc.sameSubnet, sameSubnet)
		})
	}
}

// TestUnit_InterfaceSelector_IsInterfaceUsable tests interface filter logic
func TestUnit_InterfaceSelector_IsInterfaceUsable(t *testing.T) {
	selector := ping.NewInterfaceSelector()

	testCases := []struct {
		name   string
		iface  *net.Interface
		usable bool
	}{
		{
			name:   "Nil interface",
			iface:  nil,
			usable: false,
		},
		{
			name: "Loopback interface",
			iface: &net.Interface{
				Name:  "lo",
				Flags: net.FlagUp | net.FlagLoopback,
			},
			usable: false,
		},
		{
			name: "Down interface",
			iface: &net.Interface{
				Name:  "eth0",
				Flags: 0,
			},
			usable: false,
		},
		{
			name: "Up non-loopback interface",
			iface: &net.Interface{
				Name:  "eth0",
				Flags: net.FlagUp,
			},
			usable: true,
		},
		{
			name: "Up interface with other flags",
			iface: &net.Interface{
				Name:  "docker0",
				Flags: net.FlagUp | net.FlagBroadcast,
			},
			usable: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			usable := selector.IsInterfaceUsable(tc.iface)
			assert.Equal(t, tc.usable, usable,
				"Interface %v should be usable: %v", tc.iface, tc.usable)
		})
	}
}

// TestUnit_InterfaceSelector_IPv4_EdgeCases tests edge cases for IPv4 detection
func TestUnit_InterfaceSelector_IPv4_EdgeCases(t *testing.T) {
	testCases := []struct {
		name         string
		ip           string
		shouldBeIPv4 bool
	}{
		{
			name:         "Valid IPv4",
			ip:           "192.168.1.1",
			shouldBeIPv4: true,
		},
		{
			name:         "IPv6 address",
			ip:           "2001:db8::1",
			shouldBeIPv4: false,
		},
		{
			name:         "IPv4 localhost",
			ip:           "127.0.0.1",
			shouldBeIPv4: true,
		},
		{
			name:         "IPv6 localhost",
			ip:           "::1",
			shouldBeIPv4: false,
		},
		{
			name:         "IPv4 address parsed as IPv4",
			ip:           "192.0.2.1",
			shouldBeIPv4: true,
		},
		{
			name:         "IPv4-mapped IPv6 - Go treats as IPv4 internally",
			ip:           "::ffff:192.0.2.1",
			shouldBeIPv4: true, // Go's net.ParseIP treats ::ffff:x.x.x.x as IPv4
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			require.NotNil(t, ip, "Failed to parse IP: %s", tc.ip)

			isIPv4 := ip.To4() != nil
			assert.Equal(t, tc.shouldBeIPv4, isIPv4,
				"IP %s should be IPv4: %v", tc.ip, tc.shouldBeIPv4)
		})
	}
}

// TestUnit_InterfaceSelector_ClearCache tests cache clearing
func TestUnit_InterfaceSelector_ClearCache(t *testing.T) {
	selector := ping.NewInterfaceSelector()

	// Call FindInterfaceForIPv4 before cache clear
	_, _ = selector.FindInterfaceForIPv4(net.ParseIP("127.0.0.1"))

	// Clear cache
	selector.ClearCache()

	// Call again after cache clear - should work without panic
	_, _ = selector.FindInterfaceForIPv4(net.ParseIP("127.0.0.1"))
}

// TestUnit_InterfaceSelector_CustomIPv6SubnetBits tests configurable IPv6 subnet matching
func TestUnit_InterfaceSelector_CustomIPv6SubnetBits(t *testing.T) {
	testCases := []struct {
		name       string
		ip1        string
		ip2        string
		subnetBits int
		sameSubnet bool
	}{
		{
			name:       "/64 - addresses in same /64 block",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db8:85a3:0000:ffff:ffff:ffff:ffff",
			subnetBits: 64,
			sameSubnet: true,
		},
		{
			name:       "/64 - addresses in different /64 blocks",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db8:85a4::1",
			subnetBits: 64,
			sameSubnet: false,
		},
		{
			name:       "/56 - addresses in same /56 block",
			ip1:        "2001:db8:85a3:00::1",
			ip2:        "2001:db8:85a3:ff:ffff:ffff:ffff:ffff",
			subnetBits: 56,
			sameSubnet: true,
		},
		{
			name:       "/56 - addresses in different /56 blocks",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db8:85a4::1",
			subnetBits: 56,
			sameSubnet: false,
		},
		{
			name:       "/48 - addresses in same /48 block",
			ip1:        "2001:db8:85a3:ffff:ffff:ffff:ffff:ffff",
			ip2:        "2001:db8:85a3::1",
			subnetBits: 48,
			sameSubnet: true,
		},
		{
			name:       "/48 - addresses in different /48 blocks",
			ip1:        "2001:db8:85a3::1",
			ip2:        "2001:db8:85a4::1",
			subnetBits: 48,
			sameSubnet: false,
		},
		{
			name:       "/32 - addresses in same /32 block (ISP allocation)",
			ip1:        "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff",
			ip2:        "2001:db8::1",
			subnetBits: 32,
			sameSubnet: true,
		},
		{
			name:       "/32 - addresses in different /32 blocks",
			ip1:        "2001:db8::1",
			ip2:        "2001:db9::1",
			subnetBits: 32,
			sameSubnet: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector := ping.NewInterfaceSelectorWithIPv6Bits(tc.subnetBits)
			ip1 := net.ParseIP(tc.ip1)
			ip2 := net.ParseIP(tc.ip2)
			require.NotNil(t, ip1, "Failed to parse ip1: %s", tc.ip1)
			require.NotNil(t, ip2, "Failed to parse ip2: %s", tc.ip2)

			sameSubnet := selector.IsIPv6OnSameSubnet(ip1, ip2)
			assert.Equal(t, tc.sameSubnet, sameSubnet,
				"Expected %s and %s to be on same /%d subnet: %v, got: %v",
				tc.ip1, tc.ip2, tc.subnetBits, tc.sameSubnet, sameSubnet)
		})
	}
}
