# build.ps1 - GameAP Palworld Plugin 完整构建脚本
$ErrorActionPreference = 'Stop'

Write-Host "=== GameAP Palworld Plugin 构建 ===" -ForegroundColor Cyan

# 获取脚本所在目录
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

# 1. 前端构建
Write-Host "[1/3] 安装前端依赖..." -ForegroundColor Yellow
Set-Location "$scriptDir\frontend"
pnpm install
if ($LASTEXITCODE -ne 0) { throw "pnpm install 失败" }

Write-Host "[2/3] 构建前端..." -ForegroundColor Yellow
pnpm run build
if ($LASTEXITCODE -ne 0) { throw "前端构建失败" }

# 2. WASM 构建
Write-Host "[3/3] 构建 WASM 后端..." -ForegroundColor Yellow
Set-Location $scriptDir

$env:GOOS = 'wasip1'
$env:GOARCH = 'wasm'
$env:GOCACHE = "$scriptDir\go-cache"
$env:GOPROXY = 'https://proxy.golang.org,direct'

New-Item -ItemType Directory -Force 'dist' | Out-Null

go build -buildvcs=false -buildmode=c-shared `
  -o 'dist\gameap-palworld-plugin.wasm' .

if ($LASTEXITCODE -ne 0) { throw "WASM 构建失败" }

Write-Host "=== 构建完成 ===" -ForegroundColor Green
Write-Host "输出文件: dist\gameap-palworld-plugin.wasm" -ForegroundColor Green

# 显示文件信息
Get-Item 'dist\gameap-palworld-plugin.wasm' |
  Select-Object FullName,Length,LastWriteTime
