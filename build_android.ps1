$env:CGO_ENABLED = 0
$env:GOOS = "android"
$env:GOARCH = "arm64"

$targets = @("server", "migrate", "create-user")
$outputDir = Join-Path -Path $PWD.Path -ChildPath "bin/android"

if (-not (Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir | Out-Null
    Write-Host "Created output directory: $outputDir" -ForegroundColor Green
}

foreach ($target in $targets) {
    $sourcePath = Join-Path -Path "./cmd" -ChildPath "$target"
    $outputPath = Join-Path -Path $outputDir -ChildPath "$target"

    if (-not (Test-Path $sourcePath)) {
        Write-Host "Source file not found: $sourcePath" -ForegroundColor Red
        continue
    }

    Write-Host "Building $target for Android..."
    go build -o $outputPath $sourcePath

    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully built $target for Android" -ForegroundColor Green
    } else {
        Write-Host "Failed to build $target for Android" -ForegroundColor Red
    }
}