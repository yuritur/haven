#!/usr/bin/env pwsh
# Haven installer for Windows
# Usage: irm https://raw.githubusercontent.com/yuritur/haven/master/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$repo = "yuritur/haven"

# Detect architecture
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default {
        Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
        exit 1
    }
}

# Determine install directory
$installDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $HOME ".haven\bin" }

# Fetch latest release version from GitHub API
Write-Host "Fetching latest release..." -ForegroundColor Cyan
$releaseUrl = "https://api.github.com/repos/$repo/releases/latest"
$release = Invoke-RestMethod -Uri $releaseUrl -Headers @{ "Accept" = "application/vnd.github.v3+json" }
$version = $release.tag_name -replace "^v", ""
$tag = $release.tag_name

Write-Host "Latest version: $tag" -ForegroundColor Cyan

# Build download URLs
$archiveName = "haven_${version}_windows_${arch}.zip"
$checksumsName = "haven_${version}_checksums.txt"
$baseUrl = "https://github.com/$repo/releases/download/$tag"
$archiveUrl = "$baseUrl/$archiveName"
$checksumsUrl = "$baseUrl/$checksumsName"

# Create temp directory
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "haven-install-$(Get-Random)"
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    # Download archive and checksums
    $archivePath = Join-Path $tempDir $archiveName
    $checksumsPath = Join-Path $tempDir $checksumsName

    Write-Host "Downloading $archiveName..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $archiveUrl -OutFile $archivePath -UseBasicParsing

    Write-Host "Downloading checksums..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -UseBasicParsing

    # Verify checksum
    Write-Host "Verifying checksum..." -ForegroundColor Cyan
    $expectedLine = Get-Content $checksumsPath | Where-Object { $_ -match $archiveName }
    if (-not $expectedLine) {
        Write-Error "Checksum not found for $archiveName in checksums file"
        exit 1
    }
    $expectedHash = ($expectedLine -split "\s+")[0]
    $actualHash = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()
    if ($actualHash -ne $expectedHash.ToLower()) {
        Write-Error "Checksum mismatch: expected $expectedHash, got $actualHash"
        exit 1
    }
    Write-Host "Checksum verified." -ForegroundColor Green

    # Create install directory
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    # Extract archive
    Write-Host "Extracting haven.exe to $installDir..." -ForegroundColor Cyan
    $extractDir = Join-Path $tempDir "extract"
    Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force

    # Find haven.exe (may be in a subdirectory)
    $havenExe = Get-ChildItem -Path $extractDir -Filter "haven.exe" -Recurse | Select-Object -First 1
    if (-not $havenExe) {
        Write-Error "haven.exe not found in archive"
        exit 1
    }
    Copy-Item -Path $havenExe.FullName -Destination (Join-Path $installDir "haven.exe") -Force

    # Add to PATH if not already present
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    $pathEntries = $userPath -split ";"
    if ($installDir -notin $pathEntries) {
        Write-Host "Adding $installDir to user PATH..." -ForegroundColor Cyan
        [Environment]::SetEnvironmentVariable("PATH", "$installDir;$userPath", "User")
        $env:PATH = "$installDir;$env:PATH"
        Write-Host "PATH updated. Restart your terminal for changes to take effect." -ForegroundColor Yellow
    }

    Write-Host ""
    Write-Host "Haven $tag installed successfully!" -ForegroundColor Green
    Write-Host "Binary: $(Join-Path $installDir 'haven.exe')" -ForegroundColor Green
    Write-Host ""
    Write-Host "Get started:" -ForegroundColor Cyan
    Write-Host "  haven deploy llama3.2:1b"
} finally {
    # Clean up temp directory
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
