#!/usr/bin/env bash
#
# Downloads a go-semver-release binary and verifies its SLSA provenance before making it executable, so that a
# tampered release asset is never run (the binary is then given the repository access token and GPG signing key).
#
# Requires curl, jq and slsa-verifier (https://github.com/slsa-framework/slsa-verifier).
#
# Environment:
#   GO_SEMVER_RELEASE_VERSION  release tag to download (e.g., v8.0.3), defaults to the latest release
#   GO_SEMVER_RELEASE_ASSET    asset to download, defaults to go-semver-release-linux-amd64

set -euo pipefail

repository="s0ders/go-semver-release"
asset_name="${GO_SEMVER_RELEASE_ASSET:-go-semver-release-linux-amd64}"
version="${GO_SEMVER_RELEASE_VERSION:-}"

if [ -z "${version}" ]; then
    version="$(curl -fsSL "https://api.github.com/repos/${repository}/releases/latest" | jq -er '.tag_name')"
fi

download_url="https://github.com/${repository}/releases/download/${version}"
provenance="${asset_name}.intoto.jsonl"

curl -fsSL -o "${asset_name}" "${download_url}/${asset_name}"
curl -fsSL -o "${provenance}" "${download_url}/${provenance}"

# Releases are built by the SLSA generator on pushes to the main branch, the release tag being created afterwards: the
# provenance therefore proves the binary was built by the trusted builder from the main branch of the repository.
slsa-verifier verify-artifact "${asset_name}" \
    --provenance-path "${provenance}" \
    --source-uri "github.com/${repository}" \
    --source-branch main

rm -f "${provenance}"
chmod +x "${asset_name}"

echo "Downloaded and verified ${asset_name} ${version}"
