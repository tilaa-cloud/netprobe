package ping

import (
	"net"
)

// IsIPv6 checks if the given IP address string is IPv6
func IsIPv6(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.To4() == nil && ip.To16() != nil
}

// IsIPv4 checks if the given IP address string is IPv4
func IsIPv4(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}
