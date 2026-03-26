#!/bin/bash
# SQLite database initialization script for local testing

DB_FILE="./netprobe.db"

# Create the database and schema
sqlite3 "$DB_FILE" <<EOF
CREATE TABLE IF NOT EXISTS targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    destination_ip TEXT NOT NULL UNIQUE,
    customer_id TEXT NOT NULL,
    vlan TEXT NOT NULL,
    pod TEXT NOT NULL,
    host TEXT NOT NULL,
    active INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test targets (using localhost and common IPs for testing)
INSERT OR IGNORE INTO targets (destination_ip, customer_id, vlan, pod, host, active) VALUES
    ('127.0.0.1', 'acme-corp', 'local', 'localhost', 'local-machine', 1),
    ('8.8.8.8', 'acme-corp', 'internet', 'google-dns', 'public-dns-1', 1),
    ('1.1.1.1', 'acme-corp', 'internet', 'cloudflare-dns', 'public-dns-2', 1);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_targets_active ON targets(active);
CREATE INDEX IF NOT EXISTS idx_targets_customer_id ON targets(customer_id);
EOF

echo "SQLite database initialized at $DB_FILE"
