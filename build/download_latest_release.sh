#!/bin/bash
set -euo pipefail

json_url="https://api.github.com/repos/s0ders/go-semver-release/releases/latest"
asset_name="go-semver-release-linux-amd64"

# Get the download URL
download_url=$(curl -s "$json_url" | jq -r ".assets[] | select(.name == \"$asset_name\") | .browser_download_url")

# Check if download URL was found
if [ -z "$download_url" ] || [ "$download_url" = "null" ]; then
    echo "Error: Could not find download URL for asset: $asset_name"
    exit 1
fi

echo "Downloading from: $download_url"

# Download the file
curl -L -o "$asset_name" "$download_url"

if [ $? -eq 0 ]; then
    echo "Successfully downloaded: $asset_name"
else
    echo "Error downloading file"
    exit 1
fi

chmod +x "$asset_name"