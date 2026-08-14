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
if ! CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o "${output_path}" "${source_path}"; then
    echo "Failed to build ${target} for Android" >&2
    exit 1
fi

end_time=$(date +%s%N)
duration=$(awk "BEGIN { printf \"%.2f\", (${end_time} - ${start_time}) / 1000000000 }")

echo "Successfully built ${target} for Android (${duration}s)"
