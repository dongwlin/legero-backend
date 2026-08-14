# Save original environment variables
$origCGO = $env:CGO_ENABLED
$origGOOS = $env:GOOS
$origGOARCH = $env:GOARCH

try {
    $env:CGO_ENABLED = 0
    $env:GOOS = "android"
    $env:GOARCH = "arm64"

    $target = "legero"
    $sourcePath = "./cmd/legero"
    $outputDir = Join-Path -Path $PWD.Path -ChildPath "bin/android"
    $outputPath = Join-Path -Path $outputDir -ChildPath $target

    if (-not (Test-Path $outputDir)) {
        New-Item -ItemType Directory -Path $outputDir | Out-Null
        Write-Host "Created output directory: $outputDir" -ForegroundColor Green
    }

    if (-not (Test-Path $sourcePath)) {
        Write-Host "Source file not found: $sourcePath" -ForegroundColor Red
        exit 1
    }

    Write-Host "Building $target for Android..."
    $startTime = Get-Date

    # Derive build info and inject it via ldflags; falls back to defaults
    # when git metadata is unavailable.
    $version = git describe --tags --always --dirty 2>$null
    if (-not $version) { $version = "dev" }
    $commit = git rev-parse --short HEAD 2>$null
    if (-not $commit) { $commit = "none" }
    $buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

    $ldflags = "-s -w"
    $ldflags += " -X 'github.com/dongwlin/legero-backend/internal/infra/config.Version=$version'"
    $ldflags += " -X 'github.com/dongwlin/legero-backend/internal/infra/config.Commit=$commit'"
    $ldflags += " -X 'github.com/dongwlin/legero-backend/internal/infra/config.BuildTime=$buildTime'"

    go build -trimpath --ldflags="$ldflags" -o $outputPath $sourcePath
    $endTime = Get-Date
    $duration = $endTime - $startTime

    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully built $target for Android ($([math]::Round($duration.TotalSeconds, 2))s)" -ForegroundColor Green
    } else {
        Write-Host "Failed to build $target for Android ($([math]::Round($duration.TotalSeconds, 2))s)" -ForegroundColor Red
        exit 1
    }
} finally {
    # Restore original environment variables
    $env:CGO_ENABLED = $origCGO
    $env:GOOS = $origGOOS
    $env:GOARCH = $origGOARCH
}
