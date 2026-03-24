# Foundry Tunnel Manager

Share your Foundry VTT world with players anywhere. No port forwarding needed.

## Features

- **6 tunnel providers**: Cloudflared, Playit.gg, localhost.run, Serveo, Pinggy, Tunnelmole
- **2 interfaces**: TUI, Web dashboard
- **Auto-install**: Downloads providers automatically
- **Drag & drop**: Reorder connections
- **Real-time updates**: Live status changes
- **Theming**: Multiple themes for web dashboard

**TUI shortcuts:** `↑/↓` navigate, `s` start/stop, `l` logs, `c` copy URL, `w` web, `o` open config, `a` add, `d` delete, `q` quit

## Interfaces

### TUI
![TUI](docs/tui.webp)

### Web Dashboard
![Web](docs/web.webp)

Access at `http://localhost:40500` 

> **Desktop App (Tauri)**: Temporarily removed from build pipeline. Use the TUI or Web dashboard instead.

## Installation

### Option 1: Install Script (Recommended)

```bash
curl -L https://raw.githubusercontent.com/deadbryam/ftm/main/install.sh | bash
```

This automatically detects your OS and architecture, downloads the correct binary, and installs it to `~/.local/bin`.

### Option 2: Manual Download

Download a prebuilt binary from [GitHub Releases](https://github.com/deadbryam/ftm/releases/latest).

| Platform | File |
|----------|------|
| Windows | `ftm-windows-x64.exe` |
| Linux x64 | `ftm-linux-x64` |
| Linux ARM64 | `ftm-linux-arm64` |
| macOS Intel | `ftm-macos-x64` |
| macOS Apple Silicon | `ftm-macos-arm64` |

Then run:

```bash
chmod +x ftm-*
sudo mv ftm-* /usr/local/bin/ftm
```

For macOS Apple Silicon, you may need to remove the quarantine attribute:
```bash
xattr -d com.apple.quarantine /usr/local/bin/ftm
```

### Option 3: Go

```bash
go install github.com/deadbryam/ftm@latest
```

### Run

```bash
ftm              # TUI
ftm --web        # Web dashboard only
```


## Build from Source

**Requirements:**
- Go 1.21+
- Bun 1.3+ (for building web frontend)

```bash
# Clone
git clone https://github.com/deadbryam/ftm.git
cd ftm

# Build CLI & Web
go build -o ftm ./cmd/ftm
```

## Contributing

1. Fork the repo
2. Create a branch: `git checkout -b feature/amazing`
3. Commit: `git commit -m "Add amazing feature"`
4. Push: `git push origin feature/amazing`
5. Open a Pull Request

## License

MIT License - Copyright (c) 2024 deadbryam

See [LICENSE](LICENSE) for details.
