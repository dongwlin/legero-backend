#!/usr/bin/env bash
# Build legero for Android (arm64).
# Linux/macOS equivalent of build_android.ps1.
set -euo pipefail

target="legero"
source_path="./cmd/legero"
output_dir="$(pwd)/bin/android"
output_path="${output_dir}/${target}"

if [[ ! -d "${output_dir}" ]]; then
    mkdir -p "${output_dir}"
    echo "Created output directory: ${output_dir}"
fi

if [[ ! -d "${source_path}" ]]; then
    echo "Source not found: ${source_path}" >&2
    exit 1
fi

echo "Building ${target} for Android..."
start_time=$(date +%s%N)

# Env vars are scoped to the build command only, so the caller's
# environment is left untouched (equivalent to the save/restore in
# the PowerShell script).

# Derive build info and inject it via ldflags; falls back to defaults
# when git metadata is unavailable.
version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
commit=$(git rev-parse --short HEAD 2>/dev/null || echo none)
build_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)

ldflags="-s -w"
ldflags+=" -X github.com/dongwlin/legero-backend/internal/infra/config.Version=${version}"
ldflags+=" -X github.com/dongwlin/legero-backend/internal/infra/config.Commit=${commit}"
ldflags+=" -X github.com/dongwlin/legero-backend/internal/infra/config.BuildTime=${build_time}"

if ! CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -trimpath -ldflags="${ldflags}" -o "${output_path}" "${source_path}"; then
    echo "Failed to build ${target} for Android" >&2
    exit 1
fi

end_time=$(date +%s%N)
duration=$(awk "BEGIN { printf \"%.2f\", (${end_time} - ${start_time}) / 1000000000 }")

echo "Successfully built ${target} for Android (${duration}s)"
