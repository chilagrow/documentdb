#!/bin/bash
set -e

# Change to the build directory
cd /build

# Keep the internal directory out of the Debian package
sed -i '/internal/d' Makefile

# Build the Debian package
debuild -us -uc

# Rename .deb files to include the OS name prefix
for f in ../*.deb; do mv $f $(echo $f | sed "s/\(.*\)\//\1\/$OS-/g"); done

# Create the output directory if it doesn't exist
mkdir -p /output

# Copy the built packages to the output directory
cp ../*.deb /output/

# Change ownership of the output files to match the host user's UID and GID
chown -R $(stat -c "%u:%g" /output) /output
