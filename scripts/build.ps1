# Conective Windows Build Script
# Builds both Portable Conective.exe and Single-File Conective-Setup.exe installer

$ErrorActionPreference = "Stop"
if ($PSScriptRoot) {
    Set-Location (Split-Path -Parent $PSScriptRoot)
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "     Building Conective for Windows       " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Verify Go toolchain
$goVersion = go version
Write-Host "[1/4] Go Toolchain: $goVersion" -ForegroundColor Green

# 2. Build Portable Conective.exe
Write-Host "[2/4] Compiling Conective.exe (GUI Subsystem, Strip Symbols)..." -ForegroundColor Yellow
go build -ldflags="-H=windowsgui -s -w" -o Conective.exe ./cmd/conective
if ($LASTEXITCODE -ne 0) { throw "Failed to build Conective.exe" }

# 3. Build Embedded Binaries & Assets for Installer
Write-Host "[3/4] Preparing embedded binaries and assets for installer..." -ForegroundColor Yellow
Copy-Item "Conective.exe" -Destination "cmd\installer\Conective.exe" -Force
Copy-Item "bin" -Destination "cmd\installer\bin" -Recurse -Force
Remove-Item "cmd\installer\bin\conective.exe" -Force -ErrorAction SilentlyContinue

# Package 100% CLEAN MHR runtime (from pkg/mhr/runtime: clean source only, zero personal tokens or keys)
Remove-Item "cmd\installer\bin\mhr" -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path "cmd\installer\bin\mhr" -Force | Out-Null
Copy-Item "pkg\mhr\runtime\*" -Destination "cmd\installer\bin\mhr" -Recurse -Force

# Clean any residual json configs, certificates, or bytecode
Get-ChildItem -Path "cmd\installer\bin" -Include "config.json","*.key","*.crt","*.pyc" -Recurse -Force | Remove-Item -Force -ErrorAction SilentlyContinue
Copy-Item "assets\GeoLite2-Country.mmdb" -Destination "cmd\installer\GeoLite2-Country.mmdb" -Force
Copy-Item "assets\wintun.dll" -Destination "cmd\installer\wintun.dll" -Force

# 4. Build Conective-Setup.exe Installer
Write-Host "[4/4] Compiling Conective-Setup.exe..." -ForegroundColor Yellow
go build -ldflags="-H=windowsgui -s -w" -o Conective-Setup.exe ./cmd/installer
if ($LASTEXITCODE -ne 0) { throw "Failed to build Conective-Setup.exe" }

# Verification: Ensure no private certificates or sensitive keys exist in generated binaries
$forbiddenPatterns = @(
    "BEGIN RSA PRIVATE KEY",
    "BEGIN PRIVATE KEY",
    "BEGIN EC PRIVATE KEY"
)
$secretHits = Select-String -Path "Conective.exe", "Conective-Setup.exe" -Pattern $forbiddenPatterns -SimpleMatch
if ($secretHits) {
    Remove-Item "Conective-Setup.exe" -Force -ErrorAction SilentlyContinue
    throw "SECURITY ALERT: Sensitive private keys detected in build output! Aborting build."
}

# Cleanup intermediate installer assets
Remove-Item "cmd\installer\Conective.exe" -Force -ErrorAction SilentlyContinue
Remove-Item "cmd\installer\bin" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item "cmd\installer\GeoLite2-Country.mmdb" -Force -ErrorAction SilentlyContinue
Remove-Item "cmd\installer\wintun.dll" -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Build Complete Successfully!" -ForegroundColor Green
Get-Item "Conective.exe", "Conective-Setup.exe" | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize
