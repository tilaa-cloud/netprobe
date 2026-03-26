# Packaging netprobe for AlmaLinux / RHEL

This guide explains how to build and distribute netprobe as an RPM package for AlmaLinux and RHEL systems.

## Architecture: Pre-compiled Binaries

Netprobe uses a modern packaging approach:
- **Binaries are compiled once** in the CI/CD pipeline
- **RPM build just installs** the pre-compiled binaries (no compilation)
- **Result**: Faster builds, lighter dependencies, more reproducible

This means end users only need `libpcap` at runtime, not the entire Go toolchain.

## Files Included

All RPM packaging files are located in the `rpm/` directory:

- **rpm/netprobe.spec** - RPM specification file (main packaging definition)
- **rpm/netprobe.service** - Systemd service file for the daemon
- **rpm/build-rpm.sh** - Local build script for building RPMs
- **.github/workflows/build-rpm.yml** - GitHub Actions CI/CD workflow

## Automated Build via GitHub Actions

The recommended approach is to use GitHub Actions for building RPMs. The workflow:

1. **Compiles binaries** using Go (single compile step)
2. **Creates source tarball** with pre-compiled binaries
3. **Builds RPM** that just installs the binaries (no compilation)

### Why this approach?

- ✅ **Faster RPM builds** - No Go compilation during RPM build
- ✅ **Lighter dependencies** - RPM only requires `libpcap` runtime, not build tools
- ✅ **Reproducible** - Centralized build in CI ensures consistency
- ✅ **Smaller SPEC file** - No build section code
- ✅ **Cross-compile friendly** - Can compile for different architectures in CI

### Triggering a Build

RPM builds are automatically triggered when you:

1. **Push a version tag** (recommended):
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
   This builds the RPM and automatically creates a GitHub Release with the .rpm files.

2. **Trigger manually** via GitHub Actions UI:
   - Go to Actions → Build RPM Package
   - Click "Run workflow"
   - Specify the version

### Build Artifacts

After a successful build:

- **Binary RPM**: `netprobe-1.0.0-1.el9.x86_64.rpm`
- **Source RPM**: `netprobe-1.0.0-1.el9.src.rpm`

These are available as:
- Workflow artifacts (downloadable from the Actions tab)
- GitHub Release assets (if built from a git tag)

## Building Locally

If you want to build the RPM on your machine, use the included build script:

### Quick Start
```bash
chmod +x rpm/build-rpm.sh
./rpm/build-rpm.sh 1.0.0 1
```

The script automatically:
1. Compiles the binaries locally
2. Creates a tarball with pre-compiled binaries
3. Builds the RPM package

### Prerequisites
```bash
# Build dependencies (Go toolchain for compilation step)
sudo dnf install golang git make libpcap-devel

# RPM build tools
sudo dnf install rpmdevtools rpm-build
```

### Build Process

The `rpm/build-rpm.sh` script handles everything:
```bash
./rpm/build-rpm.sh [version] [release]
```

Example:
```bash
./rpm/build-rpm.sh 1.0.0 1
```

This:
1. Compiles `netprobe` and `netprobe-ping` binaries
2. Sets up RPM build directory
3. Creates tarball with binaries (not source code)
4. Runs `rpmbuild` which just installs the binaries
5. Cleans up temporary build artifacts

Your RPM will be in:
```
~/rpmbuild/RPMS/x86_64/netprobe-1.0.0-1.el9.x86_64.rpm
```

### Manual Build (If Not Using Script)

If you prefer to build manually:

1. **Compile binaries**:
   ```bash
   go build -o netprobe ./cmd/netprobe
   go build -o netprobe-ping ./cmd/netprobe-ping
   ```

2. **Create source tarball** (with compiled binaries):
   ```bash
   mkdir -p ~/rpmbuild/SOURCES
   tar czf ~/rpmbuild/SOURCES/netprobe-1.0.0.tar.gz \
     netprobe netprobe-ping config.example.yaml netprobe.service README.md LICENSE
   ```

3. **Copy SPEC file**:
   ```bash
   cp rpm/netprobe.spec ~/rpmbuild/SPECS/
   cp rpm/netprobe.service ~/rpmbuild/SOURCES/
   ```

4. **Build RPM** (no compilation in this step):
   ```bash
   rpmbuild -ba \
     --define "version 1.0.0" \
     --define "release 1" \
     ~/rpmbuild/SPECS/netprobe.spec
   ```

## Installation

### From Local Build
```bash
sudo dnf install ~/rpmbuild/RPMS/x86_64/netprobe-1.0.0-1.el9.x86_64.rpm
```

### From GitHub Release
```bash
# Download RPM from GitHub release
wget https://github.com/your-org/netprobe/releases/download/v1.0.0/netprobe-1.0.0-1.el9.x86_64.rpm

# Install it
sudo dnf install netprobe-1.0.0-1.el9.x86_64.rpm
```

## Post-Installation

The RPM automatically:

1. **Creates system user**: `netprobe` user and group
2. **Sets capabilities**: `CAP_NET_RAW` on both binaries for raw socket access
3. **Creates directories**:
   - `/etc/netprobe/` - Configuration
   - `/var/log/netprobe/` - Log directory
   - `/var/lib/netprobe/` - State directory
4. **Installs systemd service**: Available as `systemctl start netprobe`

## Configuration

After installation, configure netprobe:

1. **Edit configuration**:
   ```bash
   sudo nano /etc/netprobe/config.yaml
   ```

2. **Start the daemon**:
   ```bash
   sudo systemctl start netprobe
   sudo systemctl enable netprobe  # Enable auto-start
   ```

3. **Check logs**:
   ```bash
   sudo journalctl -u netprobe -f
   ```

## Runtime Dependencies

After installation, netprobe only requires:

- **libpcap** - For packet capture (ARP, NDP operations)
- **glibc** - Standard C library (included with AlmaLinux)
