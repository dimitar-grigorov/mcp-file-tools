#!/usr/bin/env pwsh
# Simple test: verify MCP server builds and starts without crashing

Write-Host "Building..." -ForegroundColor Cyan
go build -o mcp-file-tools.exe .\cmd\mcp-file-tools
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed" -ForegroundColor Red; exit 1 }

Write-Host "Testing startup..." -ForegroundColor Cyan
$testDir = New-Item -ItemType Directory -Path "$env:TEMP\mcp-test" -Force

$job = Start-Job {
    param($exe, $dir)
    & $exe $dir 2>&1
} -ArgumentList (Resolve-Path ".\mcp-file-tools.exe"), $testDir.FullName

Start-Sleep -Seconds 1

if ($job.State -eq "Failed") {
    Write-Host "FAILED: Server crashed" -ForegroundColor Red
    Receive-Job $job
    exit 1
}

Write-Host "SUCCESS: Server running" -ForegroundColor Green
Stop-Job $job -ErrorAction SilentlyContinue
Remove-Job $job -Force -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force $testDir -ErrorAction SilentlyContinue
