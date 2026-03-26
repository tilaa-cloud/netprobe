%define debug_package %{nil}

Name:           netprobe
Version:        %{?version}%{!?version:1.0.0}
Release:        %{?release}%{!?release:1}%{?dist}
Summary:        Network probe tool for checking network connections

License:        MIT
URL:            https://github.com/tilaa-cloud/netprobe
# Source0 is a pre-compiled tarball built by CI/CD or build-rpm.sh script
Source0:        %{name}-%{version}.tar.gz
Source1:        %{name}.service

Requires:       libpcap
Requires(pre):  shadow-utils
Provides:       user(netprobe)
Provides:       group(netprobe)

%description
Netprobe is a network debugging tool that performs ICMP, ARP, and NDP
connectivity checks. It supports both manual testing via netprobe-ping CLI
and automatic periodic checks via the netprobe daemon with Prometheus metrics
export.

%prep
%setup -q

%build
# Binaries are pre-compiled in Source0 tarball via CI/CD or build-rpm.sh.
# No build step needed - they are installed directly in %%install section.

%check
# Pre-compiled binaries, no tests in RPM build

%install
# Create necessary directories
install -d %{buildroot}%{_sbindir}
install -d %{buildroot}%{_bindir}
install -d %{buildroot}%{_unitdir}
install -d %{buildroot}%{_sysconfdir}/netprobe
install -d %{buildroot}%{_localstatedir}/log/netprobe
install -d %{buildroot}%{_localstatedir}/lib/netprobe

# Install binaries
install -m 0755 netprobe %{buildroot}%{_sbindir}/netprobe
install -m 0755 netprobe-ping %{buildroot}%{_bindir}/netprobe-ping

# Install systemd service file
install -m 0644 %{SOURCE1} %{buildroot}%{_unitdir}/netprobe.service

# Install default configuration
install -m 0640 config.example.yaml %{buildroot}%{_sysconfdir}/netprobe/config.yaml.example

# Create default empty config (user must edit it)
cat > %{buildroot}%{_sysconfdir}/netprobe/config.yaml <<'EOF'
# Netprobe Configuration
# See config.yaml.example for complete documentation

exporter:
  listen_address: "0.0.0.0"
  listen_port: 9090

scheduler:
  ping_interval_seconds: 60
  batch_size: 100
  max_parallel_workers: 10

icmp:
  timeout_ms: 5000
  count: 1

arp:
  timeout_ms: 5000

ndp:
  timeout_ms: 5000

database:
  # Uncomment and configure one of the following:
  # type: "postgresql"
  # host: "localhost"
  # port: 5432
  # database: "network_monitoring"
  # user: "netprobe"
  # password: "changeme"
  # query: "SELECT destination_ip, customer_id, vlan, pod, host FROM targets WHERE active = true ORDER BY destination_ip"
  # dimension_labels:
  #   - customer_id
  #   - vlan
  #   - pod
  #   - host
  type: ""
EOF
chmod 0640 %{buildroot}%{_sysconfdir}/netprobe/config.yaml

%pre
# Create netprobe user and group
getent group netprobe >/dev/null || groupadd -r netprobe
getent passwd netprobe >/dev/null || useradd -r -g netprobe -s /sbin/nologin -d /var/lib/netprobe -m netprobe

%post
# Set CAP_NET_RAW capability for raw socket access (required for ARP, NDP, and ICMP operations)
/usr/sbin/setcap cap_net_raw=ep /usr/sbin/netprobe || true
/usr/sbin/setcap cap_net_raw=ep /usr/bin/netprobe-ping || true

# Set proper ownership and permissions
chown -R netprobe:netprobe %{_sysconfdir}/netprobe
chown -R netprobe:netprobe %{_localstatedir}/log/netprobe
chown -R netprobe:netprobe %{_localstatedir}/lib/netprobe

# Reload systemd daemon to recognize new service
systemctl daemon-reload >/dev/null 2>&1 || true

%preun
# Stop service before uninstall
if [ $1 -eq 0 ]; then
  systemctl --no-reload disable netprobe >/dev/null 2>&1 || true
  systemctl stop netprobe >/dev/null 2>&1 || true
fi

%postun
# Reload systemd daemon
systemctl daemon-reload >/dev/null 2>&1 || true

%files
%attr(0755, root, root) %{_sbindir}/netprobe
%attr(0755, root, root) %{_bindir}/netprobe-ping
%{_unitdir}/netprobe.service
%attr(0640, netprobe, netprobe) %config(noreplace) %{_sysconfdir}/netprobe/config.yaml
%attr(0640, root, root) %{_sysconfdir}/netprobe/config.yaml.example
%attr(0755, netprobe, netprobe) %dir %{_localstatedir}/log/netprobe
%attr(0755, netprobe, netprobe) %dir %{_localstatedir}/lib/netprobe

%changelog
* Wed Mar 25 2026 Netprobe Team <netprobe@tilaa.com> - 1.0.0-1
- Initial RPM package
