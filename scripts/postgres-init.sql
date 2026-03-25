-- PostgreSQL initialization script for pingding
CREATE TABLE
    IF NOT EXISTS targets (
        id SERIAL PRIMARY KEY,
        destination_ip VARCHAR(255) NOT NULL UNIQUE,
        customer_id VARCHAR(100) NOT NULL,
        vlan VARCHAR(50) NOT NULL,
        pod VARCHAR(100) NOT NULL,
        host VARCHAR(255) NOT NULL,
        active BOOLEAN DEFAULT true,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- Insert test targets - using actual IP addresses assigned in docker-compose
INSERT INTO
    targets (
        destination_ip,
        customer_id,
        vlan,
        pod,
        host,
        active
    )
VALUES
    -- IPv4 targets
    (
        '172.20.0.100',
        'acme-corp',
        'production',
        'us-west-2',
        'server-01',
        true
    ),
    (
        '172.20.0.101',
        'acme-corp',
        'production',
        'us-west-2',
        'server-02',
        true
    ),
    (
        '172.20.0.102',
        'acme-corp',
        'staging',
        'us-east-1',
        'test-server-01',
        true
    ),
    -- IPv6 targets
    (
        'fd00::100',
        'acme-corp',
        'production',
        'us-west-2',
        'server-ipv6-01',
        true
    ),
    (
        'fd00::101',
        'acme-corp',
        'production',
        'us-west-2',
        'server-ipv6-02',
        true
    ),
    (
        'fd00::102',
        'acme-corp',
        'staging',
        'us-east-1',
        'test-server-ipv6-01',
        true
    ),
    (
        'fd00::200',
        'beta-corp',
        'production',
        'eu-west-1',
        'ipv6-only-01',
        true
    ),
    (
        'fd00::201',
        'beta-corp',
        'staging',
        'eu-west-1',
        'ipv6-only-02',
        true
    ) ON CONFLICT (destination_ip) DO NOTHING;

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_targets_active ON targets (active);

CREATE INDEX IF NOT EXISTS idx_targets_customer_id ON targets (customer_id);