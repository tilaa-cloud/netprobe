# Netprobe

Netprobe is a toolset that can be used for debugging networking issues. The
netprobe-ping binary can test ICMP traffic to both IPv4 and IPv6 addresses,
perform ARP ping to IPv4 addresses, and perform Neighbour Solicitation on IPv6
addresses.

The daemon binary (netprobe) will execute those checks automatically, and expose
the results as a Prometheus exporter.

## Features

- **Multiple Ping Methods**: Supports both ICMP and ARP ping protocols out of the box
- **Scalable Design**: Processes targets in parallel batches to handle thousands of endpoints
- **Pluggable Target Sources**: Database-driven targets with extensible source architecture
- **Prometheus Integration**: Exposes metrics in standard Prometheus format
- **Rich Metrics**: Tracks packet loss percentage and min/max/average latency per target
- **Dimensional Analysis**: Labels for customer_id, VLAN, pod, and host for anomaly detection

## Building

```bash
make build
```

## Configuration

Create a `config.yaml` based on `config.example.yaml`:

```yaml
exporter:
  listen_address: "0.0.0.0"
  listen_port: 9090
  ping_interval_seconds: 60
  batch_size: 100
  max_parallel_workers: 10
  icmp:
    enabled: true
    timeout_ms: 5000
    count: 1
  arp:
    enabled: true
    timeout_ms: 5000

database:
  type: "postgresql"
  host: "localhost"
  port: 5432
  database: "network_db"
  user: "exporter"
  password: "${DB_PASSWORD}"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime_seconds: 600
```

## Running

```bash
./pingding --config config.yaml
```

Metrics will be available at `http://localhost:9090/metrics`

## Running Tests

```bash
make test
```

## Metrics

### Packet Loss
- **Name**: `pingding_packet_loss_percent`
- **Type**: Gauge
- **Labels**: `destination_ip`, `method`, `customer_id`, `vlan`, `pod`, `host`
- **Value**: 0-100 (percentage of lost packets)

### Latency (Min/Max/Avg)
- **Names**: `pingding_latency_min_ms`, `pingding_latency_max_ms`, `pingding_latency_avg_ms`
- **Type**: Gauge
- **Labels**: Same as packet loss
- **Value**: Response time in milliseconds

## Example Metrics

```
pingding_packet_loss_percent{destination_ip="10.0.0.1",method="icmp",customer_id="acme",vlan="prod",pod="us-west-2",host="server-01"} 0.0
pingding_latency_avg_ms{destination_ip="10.0.0.1",method="icmp",customer_id="acme",vlan="prod",pod="us-west-2",host="server-01"} 2.5
```

## Architecture

The exporter uses a scalable, concurrent architecture:

1. **Target Fetching**: Periodically loads targets from configured database source
2. **Batch Scheduling**: Divides targets into batches to prevent resource exhaustion
3. **Parallel Execution**: Workers process multiple targets concurrently within each batch
4. **Non-blocking Collection**: Results collected concurrently to prevent deadlocks
5. **Metrics Storage**: Thread-safe in-memory metrics storage
6. **HTTP Exposition**: Prometheus-compatible `/metrics` endpoint
