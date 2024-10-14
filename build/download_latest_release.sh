#!/bin/bash

json_url="https://api.github.com/repos/s0ders/go-semver-release/releases/latest"
asset_name="go-semver-release-linux-amd64"

# Get the download URL
api_response=$(curl -s --no-progress-meter "$json_url")
download_url=$(echo "$api_response" | jq -r ".assets[] | select(.name == \"$asset_name\") | .browser_download_url")
asset_version=$(echo "$api_response" | jq -r ".tag_name")

# Check if download URL was found
if [ -z "$download_url" ] || [ "$download_url" = "null" ]; then
    echo "Error: Could not find download URL for asset: $asset_name"
    exit 1
fi

# Download the file
curl -L -o "$asset_name" "$download_url"

if [ $? -eq 0 ]; then
    echo "Successfully downloaded: $asset_name $asset_version"
else
    echo "Error downloading file"
    exit 1
fi

chmod +x "$asset_name"