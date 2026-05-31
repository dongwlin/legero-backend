$env:CGO_ENABLED = 0
$env:GOOS = "android"
$env:GOARCH = "arm64"

$targets = @("server", "create-user")
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
    $startTime = Get-Date
    go build -trimpath --ldflags='-s -w' -o $outputPath $sourcePath
    $endTime = Get-Date
    $duration = $endTime - $startTime

    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully built $target for Android ($([math]::Round($duration.TotalSeconds, 2))s)" -ForegroundColor Green
    } else {
        Write-Host "Failed to build $target for Android ($([math]::Round($duration.TotalSeconds, 2))s)" -ForegroundColor Red
    }
}