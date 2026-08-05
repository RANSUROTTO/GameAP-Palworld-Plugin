# GameAP Palworld REST Control

English | [中文](README_zh.md)

A native Palworld server management tab for GameAP. Uses WASM backend to proxy the official Palworld REST API, so passwords never reach the browser.

## Screenshots

| Server | Players | Announce |
|:---:|:---:|:---:|
| ![Server](docs/images/server.png) | ![Players](docs/images/players.png) | ![Announce](docs/images/announce.png) |

| Player Management | World Settings |
|:---:|:---:|
| ![Player Management](docs/images/player-mgmt.png) | ![World Settings](docs/images/world.png) |

## Features

- View server status, online players, performance metrics
- Kick, ban, unban players
- Send announcements
- Save world, graceful shutdown, force stop
- Independent REST config per server instance
- Admin-only operations

## Requirements

- Go 1.26+
- Node.js 20 or 22
- pnpm
- Docker Desktop (with WSL 2 enabled)

## Project Structure

```
GameAP-Palworld-Plugin\
├─ frontend\              # Vue frontend
│  ├─ src\
│  │  ├─ index.ts
│  │  └─ PalworldTab.vue
│  ├─ dist\               # Frontend build output
│  ├─ package.json
│  └─ vite.config.js
├─ main.go                # WASM backend
├─ go.mod
├─ go.sum
├─ dist\                  # WASM build output
│  └─ gameap-palworld-plugin.wasm
├─ docs\                  # Screenshots
├─ build.ps1              # One-click build script
└─ README.md
```

## Build

### One-click build

```powershell
.\build.ps1
```

### Manual build

Install frontend dependencies:

```powershell
cd frontend
pnpm install
```

If pnpm reports esbuild error:

```powershell
node node_modules\esbuild\install.js
```

Build frontend:

```powershell
pnpm run build
```

Build WASM:

```powershell
cd ..
$env:GOOS = 'wasip1'
$env:GOARCH = 'wasm'
$env:GOCACHE = "$(pwd)\go-cache"
$env:GOPROXY = 'https://proxy.golang.org,direct'

New-Item -ItemType Directory -Force 'dist' | Out-Null

go build -buildvcs=false -buildmode=c-shared -o 'dist\gameap-palworld-plugin.wasm' .
```

If you can't access the official Go Proxy, use a mirror:

```powershell
$env:GOPROXY = 'https://goproxy.cn,direct'
```

Verify the build output:

```powershell
Get-Item 'dist\gameap-palworld-plugin.wasm'
```

## Deploy

### Install via GameAP UI

1. Login to GameAP
2. Admin → Plugins → Upload/Install
3. Select `dist\gameap-palworld-plugin.wasm`
4. Click Install

### Install via API

```powershell
$gameapUrl = 'http://localhost:8025'
$gameapAdmin = 'admin'
$gameapPassword = 'your-gameap-password'
$wasm = 'dist\gameap-palworld-plugin.wasm'

# Login to get token
$auth = Invoke-RestMethod -Method Post -Uri "$gameapUrl/api/auth/login" `
  -ContentType 'application/json' `
  -Body (@{login=$gameapAdmin;password=$gameapPassword} | ConvertTo-Json)
$token = if ($auth.token) {$auth.token} elseif ($auth.data.token) {$auth.data.token} else {$auth.access_token}

# Upload plugin
curl.exe -sS -X POST `
  -H "Authorization: Bearer $token" `
  -F "file=@$wasm;type=application/wasm" `
  "$gameapUrl/api/admin/plugins/upload/install"
```

### Configure REST Connection

After installing the plugin, configure the Palworld REST connection:

```powershell
$headers = @{Authorization="Bearer $token"}
$pluginId = 'h4fkd4cukuk6o'  # Check actual ID in plugin management page
$serverId = 1
$palPassword = 'your-palworld-admin-password'

$config = @{
  base_url = 'http://host.docker.internal:8212/v1/api'
  username = 'admin'
  password = $palPassword
} | ConvertTo-Json

# Write config
Invoke-RestMethod -Method Post `
  -Uri "$gameapUrl/api/plugins/$pluginId/servers/$serverId/config" `
  -Headers $headers -ContentType 'application/json' -Body $config

# Verify password is not returned
Invoke-RestMethod `
  -Uri "$gameapUrl/api/plugins/$pluginId/servers/$serverId/config" `
  -Headers $headers
```

Response should only contain `configured`, `base_url`, `username` - no password.

## GameAP Container Configuration

Add to `docker-compose.local.yml`:

```yaml
services:
  gameap:
    environment:
      PLUGIN_HTTP_ALLOWED_SCHEMES: http,https
      PLUGIN_HTTP_ALLOWED_HOSTS: host.docker.internal
```

Only allow `host.docker.internal`, do not open internal network access.

## API Endpoints

Read:

```powershell
$base = "$gameapUrl/api/plugins/$pluginId/servers/$serverId"

Invoke-RestMethod "$base/info" -Headers $headers      # Server info
Invoke-RestMethod "$base/players" -Headers $headers    # Online players
Invoke-RestMethod "$base/settings" -Headers $headers   # Settings
Invoke-RestMethod "$base/metrics" -Headers $headers    # Metrics
```

Management:

```powershell
# Send announcement
Invoke-RestMethod -Method Post -Uri "$base/announce" `
  -Headers $headers -ContentType 'application/json' `
  -Body (@{message='Test announcement'} | ConvertTo-Json)

# Save world
Invoke-RestMethod -Method Post -Uri "$base/save" -Headers $headers

# Kick player (use userId, not nickname)
$userId = 'steam_00000000000000000'
Invoke-RestMethod -Method Post -Uri "$base/kick" `
  -Headers $headers -ContentType 'application/json' `
  -Body (@{userid=$userId;message='Violation'} | ConvertTo-Json)

# Ban player
Invoke-RestMethod -Method Post -Uri "$base/ban" `
  -Headers $headers -ContentType 'application/json' `
  -Body (@{userid=$userId;message='Violation'} | ConvertTo-Json)

# Unban player
Invoke-RestMethod -Method Post -Uri "$base/unban" `
  -Headers $headers -ContentType 'application/json' `
  -Body (@{userid=$userId} | ConvertTo-Json)
```

Server control:

```powershell
# Graceful shutdown (stops GameAP service wrapper)
Invoke-RestMethod -Method Post -Uri "$base/shutdown" -Headers $headers

# Force stop
Invoke-RestMethod -Method Post -Uri "$base/stop" -Headers $headers

# Start server
Invoke-RestMethod -Method Post -Uri "$gameapUrl/api/servers/$serverId/start" -Headers $headers
```

Graceful shutdown and force stop require GameAP service coordination, do not call Palworld REST directly.

## Troubleshooting

### Clicking plugin causes GameAP logout

When Palworld REST returns 401/403, the WASM backend converts it to 502, which won't trigger GameAP logout. If you experience this, check if the frontend is calling Palworld REST directly.

### Buttons not responding

- Announcement cannot be empty
- Kick/ban must select online players
- Unban requires full `userId`
- Refresh page to check for errors
- Browser `Ctrl+F5` to clear cache

### Server restarts after shutdown

This is caused by the Shawl wrapper auto-restart. Correct flow:

1. Call Palworld `/shutdown`
2. Immediately call GameAP `/api/servers/{id}/stop`
3. Wait for service to stop and process count to be 0

### World snapshot returns 501

Current Palworld version doesn't support this feature, plugin returns 501. Other features work normally.

### esbuild installation failed

```powershell
cd frontend
node node_modules\esbuild\install.js
node node_modules\vite\bin\vite.js build
```

### Go cache issues

```powershell
$env:GOCACHE = "$(pwd)\go-cache"
go clean -cache
go build -buildvcs=false -buildmode=c-shared -o 'dist\gameap-palworld-plugin.wasm' .
```

## Update Plugin

After modifying frontend or backend code:

```powershell
# Build
.\build.ps1

# Uninstall old plugin in GameAP
# Upload new wasm
# Reconfigure REST connection
# Restart container
docker restart gameap

# Browser Ctrl+F5
```

## Debugging

View logs:

```powershell
docker logs gameap --tail 100 -f
docker logs gameap 2>&1 | Select-String -Pattern "plugin|palworld"
```

Test Palworld REST connectivity:

```powershell
$pair = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("admin:your-password"))
Invoke-RestMethod 'http://localhost:8212/v1/api/info' -Headers @{Authorization="Basic $pair"}
```

Check installed plugins:

```powershell
Invoke-RestMethod "$gameapUrl/api/admin/plugins/loaded" -Headers $headers
```

## License

MIT
