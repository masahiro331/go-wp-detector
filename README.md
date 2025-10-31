# go-wp-detector

WordPress plugin name and version detector with vulnerability scanning using WPScan API.

## Features

- **Plugin Detection**: Detect WordPress plugin names and versions from plugin files
- **Vulnerability Scanning**: Scan for known vulnerabilities using WPScan API
- **Test Data Downloader**: Download popular WordPress plugins for testing

## Project Status

###  Completed

- WordPress.org API client implementation
- Plugin download script (`cmd/download-plugins`)
- Architecture and WPScan API documentation

### =§ In Progress

- WPScan API client implementation
- Plugin detector implementation

## Quick Start

### Download Test Plugins

Download the top 100 most popular WordPress plugins:

```bash
go run cmd/download-plugins/main.go -count 100
```

Options:
- `-count N`: Number of plugins to download (default: 100)
- `-output DIR`: Output directory (default: testdata/wp-content/plugins)

### Run Tests

```bash
go test ./...
```

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

### Components

- `pkg/wordpress`: WordPress.org API client for querying and downloading plugins
- `pkg/wpscan`: WPScan API client for vulnerability scanning (coming soon)
- `pkg/detector`: Plugin name/version detector (coming soon)
- `cmd/download-plugins`: CLI tool for downloading test data

## WPScan API

See [docs/WPSCAN_API.md](docs/WPSCAN_API.md) for WPScan API investigation and usage plan.

To use WPScan API:
1. Register at https://wpscan.com/
2. Get API token from https://wpscan.com/profile
3. Set environment variable: `export WPSCAN_API_TOKEN=your_token`

## Development

This project follows TDD (Test-Driven Development) methodology.

```bash
# Run tests
go test -v ./pkg/wordpress

# Format code
go fmt ./...

# Download test plugins
go run cmd/download-plugins/main.go -count 5
```

## License

MIT
