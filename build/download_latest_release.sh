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
#   GITHUB_TOKEN               optional token authenticating the GitHub API request resolving the latest release

set -euo pipefail

repository="s0ders/go-semver-release"
asset_name="${GO_SEMVER_RELEASE_ASSET:-go-semver-release-linux-amd64}"
version="${GO_SEMVER_RELEASE_VERSION:-}"

if [ -z "${version}" ]; then
    api_url="https://api.github.com/repos/${repository}/releases/latest"
    # Unauthenticated API requests are rate limited per IP address, which runners share: authenticate when a token is
    # available. The header is read from stdin so that the token never appears on a command line.
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        release="$(printf 'Authorization: Bearer %s\n' "${GITHUB_TOKEN}" | curl -fsSL -H @- "${api_url}")"
    else
        release="$(curl -fsSL "${api_url}")"
    fi
    version="$(jq -er '.tag_name' <<< "${release}")"
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
