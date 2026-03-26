#!/bin/bash
# rpm/build-rpm.sh - Local RPM build script for netprobe
# Builds binaries locally, then packages into RPM (no compilation during RPM build)
# Usage: ./rpm/build-rpm.sh [version] [release]
# Or from rpm/: ./build-rpm.sh [version] [release]

set -e

VERSION="${1:-1.0.0}"
RELEASE="${2:-1}"

echo "Building netprobe RPM"
echo "Version: $VERSION"
echo "Release: $RELEASE"

# Check prerequisites
command -v rpmbuild >/dev/null 2>&1 || {
    echo "ERROR: rpmbuild not found. Install with: sudo dnf install rpmdevtools rpm-build"
    exit 1
}

command -v go >/dev/null 2>&1 || {
    echo "ERROR: Go not found. Install with: sudo dnf install golang"
    exit 1
}

# Set up RPM build structure if not already done
if [ ! -d "$HOME/rpmbuild/SPECS" ]; then
    echo "Setting up RPM build structure..."
    rpmdev-setuptree
fi

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Go to repo root (one level up from rpm/ directory)
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$REPO_ROOT"

# Pre-compile binaries
echo "Compiling binaries in $REPO_ROOT..."
go build -o netprobe ./cmd/netprobe
go build -o netprobe-ping ./cmd/netprobe-ping
echo "✓ Binaries compiled"

# Create source tarball with pre-compiled binaries
echo "Creating source tarball with pre-compiled binaries..."
SOURCES_DIR="$HOME/rpmbuild/SOURCES"
TARBALL="$SOURCES_DIR/netprobe-$VERSION.tar.gz"

# Create temporary staging directory
STAGE_DIR=$(mktemp -d)
PKG_DIR="$STAGE_DIR/netprobe-$VERSION"
mkdir -p "$PKG_DIR"

# Copy files into staging directory
cp "$REPO_ROOT/netprobe" "$PKG_DIR/"
cp "$REPO_ROOT/netprobe-ping" "$PKG_DIR/"
cp "$REPO_ROOT/Makefile" "$PKG_DIR/"
cp "$REPO_ROOT/README.md" "$PKG_DIR/"
cp "$REPO_ROOT/LICENSE" "$PKG_DIR/"
cp "$REPO_ROOT/config.example.yaml" "$PKG_DIR/"
cp "$SCRIPT_DIR/netprobe.service" "$PKG_DIR/"
cp -r "$REPO_ROOT/scripts" "$PKG_DIR/" 2>/dev/null || true

# Create tarball from staging directory
tar -czf "$TARBALL" -C "$STAGE_DIR" "netprobe-$VERSION" || {
    echo "Tarball creation failed"
    rm -rf "$STAGE_DIR"
    exit 1
}

# Clean up staging directory
rm -rf "$STAGE_DIR"

echo "Source tarball created: $TARBALL"

# Copy SPEC and service files
echo "Copying build files..."
cp "$SCRIPT_DIR/netprobe.spec" "$HOME/rpmbuild/SPECS/"
cp "$SCRIPT_DIR/netprobe.service" "$SOURCES_DIR/"

# Build RPM
echo "Building RPM (no compilation step needed)..."
rpmbuild -ba \
    --define "version $VERSION" \
    --define "release $RELEASE" \
    "$HOME/rpmbuild/SPECS/netprobe.spec"

# Clean up local binaries
rm -f "$REPO_ROOT/netprobe" "$REPO_ROOT/netprobe-ping"

# Display results
echo ""
echo "=========================================="
echo "✅ Build complete!"
echo "=========================================="
echo ""

# Find and display binary RPM
BINARY_RPM=$(find "$HOME/rpmbuild/RPMS/x86_64" -name "netprobe-$VERSION-$RELEASE*.x86_64.rpm" -type f 2>/dev/null | head -1)
if [ -n "$BINARY_RPM" ]; then
    echo "📦 Binary RPM:"
    echo "  $(basename "$BINARY_RPM")"
else
    echo "❌ Binary RPM not found"
fi

echo ""

# Find and display source RPM (if it exists)
SOURCE_RPM=$(find "$HOME/rpmbuild/SRPMS" -name "netprobe-$VERSION-$RELEASE*.src.rpm" -type f 2>/dev/null | head -1)
if [ -n "$SOURCE_RPM" ]; then
    echo "📦 Source RPM:"
    echo "  $(basename "$SOURCE_RPM")"
else
    echo "ℹ️  Source RPM not found (optional)"
fi

echo ""
echo "🚀 Installation:"
if [ -n "$BINARY_RPM" ]; then
    echo "  sudo dnf install $BINARY_RPM"
else
    echo "  sudo dnf install /path/to/netprobe-*.x86_64.rpm"
fi
echo ""
echo "🔍 Check package:"
echo "  rpmlint $HOME/rpmbuild/SPECS/netprobe.spec"
echo ""
