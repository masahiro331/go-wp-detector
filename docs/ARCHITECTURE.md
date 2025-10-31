# go-wp-detector Architecture

## Overview

go-wp-detector is a tool for detecting WordPress plugin names and versions, and scanning for vulnerabilities using the WPScan API.

## Components

### 1. WordPress Plugin API Client (`pkg/wordpress`)

Responsible for interacting with the WordPress.org Plugin API.

**Responsibilities:**
- Query plugins by popularity, downloads, or other criteria
- Fetch plugin metadata (name, version, download URL, etc.)
- Download plugin ZIP files

**API Endpoint:**
- Base URL: `https://api.wordpress.org/plugins/info/1.2/`
- Query endpoint: `?action=query_plugins&request[browse]=popular&request[per_page]=N&request[page]=M`

**Key Data Structure:**
```go
type PluginInfo struct {
    Name         string   `json:"name"`
    Slug         string   `json:"slug"`
    Version      string   `json:"version"`
    DownloadLink string   `json:"download_link"`
    Rating       float64  `json:"rating"`
    ActiveInstalls int    `json:"active_installs"`
    Downloaded   int      `json:"downloaded"`
    // ... other fields
}
```

### 2. WPScan API Client (`pkg/wpscan`)

Responsible for interacting with the WPScan Vulnerability Database API.

**Responsibilities:**
- Query plugin vulnerabilities by slug
- Authenticate with API token
- Parse vulnerability data

**API Endpoint:**
- Base URL: `https://wpscan.com/api/v3/`
- Plugin vulnerabilities: `/plugins/{slug}`
- Plugin version-specific: `/plugins/{slug}/{version}`

**Authentication:**
- Header: `Authorization: Token token=API_TOKEN`
- Free tier: 25 requests/day
- API token from: https://wpscan.com/profile

**Key Data Structure:**
```go
type VulnerabilityReport struct {
    Slug            string          `json:"slug"`
    Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    FixedIn     string   `json:"fixed_in"`
    References  []string `json:"references"`
    // ... other fields
}
```

### 3. Plugin Detector (`pkg/detector`)

Responsible for detecting plugin name and version from plugin files.

**Responsibilities:**
- Parse plugin main file headers
- Extract plugin name and version
- Support various plugin file structures

**Detection Strategy:**
WordPress plugins contain a main PHP file with headers like:
```php
/**
 * Plugin Name: Example Plugin
 * Version: 1.2.3
 * ...
 */
```

The detector will:
1. Search for PHP files in the plugin directory
2. Parse file headers to extract metadata
3. Return plugin name and version

### 4. Download Script (`cmd/download-plugins`)

CLI tool for downloading popular WordPress plugins for testing.

**Features:**
- Download N most popular plugins from WordPress.org
- Extract ZIP files to `testdata/wp-content/plugins/`
- Maintain WordPress directory structure
- Support configurable download count

**Usage:**
```bash
go run cmd/download-plugins/main.go -count 100
```

## Data Flow

### Plugin Download Flow
```
User -> download-plugins CLI
  -> WordPress API Client
  -> Download ZIP files
  -> Extract to testdata/wp-content/plugins/
```

### Vulnerability Scan Flow
```
User -> Detector (scans plugin directory)
  -> Extract plugin name & version
  -> WPScan API Client (queries vulnerabilities)
  -> Return vulnerability report
```

## Directory Structure

```
go-wp-detector/
├── cmd/
│   └── download-plugins/    # CLI tool for downloading test data
├── pkg/
│   ├── wordpress/           # WordPress.org API client
│   ├── wpscan/             # WPScan API client
│   └── detector/           # Plugin name/version detector
├── testdata/
│   └── wp-content/
│       └── plugins/        # Downloaded plugins for testing
└── docs/                   # Documentation
```

## Testing Strategy

### Unit Tests
- Each package (`wordpress`, `wpscan`, `detector`) has its own tests
- Mock external API calls using interfaces
- Table-driven tests for various scenarios

### Integration Tests
- Use downloaded plugins in `testdata/` for realistic testing
- Test detector against real plugin files
- Test WPScan integration with sample data

## Implementation Phases

### Phase 1: Download Script (Current)
1. Implement WordPress API client
2. Create download-plugins CLI tool
3. Download 100 popular plugins for testing

### Phase 2: WPScan Investigation
1. Document WPScan API usage
2. Create API access plan
3. Design WPScan client interface

### Phase 3: Detector Implementation
1. Design detector interface
2. Implement plugin header parser
3. Test with downloaded plugins

### Phase 4: Vulnerability Scanner
1. Implement WPScan client
2. Integrate detector with WPScan
3. Create scanner CLI tool
