# RPM Packaging Files

This directory contains all files needed to build netprobe as an RPM package for AlmaLinux and RHEL systems.

## Files

- **netprobe.spec** - RPM specification file defining the package
- **netprobe.service** - Systemd service file for the netprobe daemon
- **build-rpm.sh** - Build script for creating RPMs locally

## Quick Start

### Build an RPM locally:

```bash
chmod +x build-rpm.sh
./build-rpm.sh 1.0.0 1
```

The RPM will be created in `~/rpmbuild/RPMS/x86_64/`

### Prerequisites

```bash
sudo dnf install golang git rpmdevtools rpm-build libpcap-devel
```

## Details

See [../PACKAGING.md](../PACKAGING.md) for complete documentation on RPM packaging, automated builds via GitHub Actions, and installation instructions.

## Building from Repository Root

You can also run the build script from the repository root:

```bash
./rpm/build-rpm.sh 1.0.0 1
```
