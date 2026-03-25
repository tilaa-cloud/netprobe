package target

// Target represents an IP endpoint to be monitored
type Target struct {
	DestinationIP string            // The IP address to ping
	Dimensions    map[string]string // Custom dimensions (e.g., customer_id, vlan, pod, host)
}
