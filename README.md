# RDP Portal

A slightly smarter and simple Remote Desktop manager for Windows.

<img src="https://user-images.githubusercontent.com/1336778/198676751-f4e21e04-ee22-4418-a28c-c2f0704b6764.png" width="600" />

## Why RDP Portal?

By default, Remote Desktop Connection (mstsc.exe) is not possible to backup the saved passwords and the connection management is not good.

With RDP Portal, it is possible to save password and use it on any other Windows PCs.

## ⭐ Features

- Portable
- Backup-able Saved Passwords
- Same user experience while connected to your remote desktop.
- The workflow is similar to WinSCP or HeidiSQL

## Download

From GitHub: https://github.com/louislam/rdp-portal/releases/latest

## FAQ

1. How to Backup?

> RDP Portal is a portable application, you just need to backup the folder.

2. How to config more settings like screen resolution?

> You can click `Options` in RDP Portal, it will open the original Remote Desktop Connection application. You can set anything there, but you have to click `Save` at the end. Don't click `Connect`.

<img src="https://user-images.githubusercontent.com/1336778/198681115-c3ccb823-43b8-480a-b81c-ff32d5d117ae.png" width=600 />

## Development

Go 1.24+, Node.js, and the [Wails v3](https://v3.wails.io) CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
wails3 dev      # hot reload: Vue/TS edits reload in place, Go edits rebuild
wails3 build    # -> bin/RDP-Portal.exe
```

Stack: Go + Wails v3, Vue 3 (Options API), Bootstrap 6.0.0-alpha.1.

| Path | Contents |
| --- | --- |
| `internal/core` | config.json, profiles, .rdp generation, DES/DPAPI |
| `app.go` | Wails service exposed to the frontend |
| `frontend/src/App.vue` | UI |
| `build/` | Windows icon, manifest, NSIS/MSIX templates |

## Compatibility with 1.x (C#)

`config.json` keeps the same shape, the password encryption keeps the same DES key/IV, and
the `password 51:b:` value still comes from DPAPI (`CryptProtectData`, description `psw`),
so an existing portable folder and its saved passwords keep working.

## Tests

```bash
go test ./... -count=1 -cover     # core + service layer (77% statements)
cd frontend && npm test           # Vue component tests (jsdom, bindings mocked)
node e2e/run.mjs                  # real exe + WebView2 over the devtools protocol
```

`e2e/run.mjs` launches `bin/RDP-Portal.exe` from a throwaway folder, drives the actual
window (CDP via `RDP_PORTAL_CDP_PORT`) and checks the UI plus the `config.json` /
`rdp-files/` it writes. Build first: `wails3 build`.

## Credit

- <a href="https://www.flaticon.com/free-icons/computer" title="computer icons">Computer icons created by Freepik - Flaticon</a>

