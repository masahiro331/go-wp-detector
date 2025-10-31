# Detector Implementation Plan

## Overview

Implement a WordPress plugin detector that identifies plugin names and versions from plugin files, following WordPress's `get_plugins()` behavior.

## Architecture

```
pkg/detector/
├── scanner.go         # Directory scanning logic
├── scanner_test.go    # Scanner tests
├── parser.go          # Plugin header parsing
├── parser_test.go     # Parser tests
├── detector.go        # Main detector interface
└── detector_test.go   # Integration tests
```

## Implementation Phases

### Phase 1: Plugin Header Parser (TDD)

**Goal**: Parse plugin headers from PHP file content

#### Data Structures

```go
package detector

// PluginInfo represents detected plugin metadata
type PluginInfo struct {
    Name              string // Required: Plugin Name
    Version           string // Critical: Version
    Slug              string // Derived from path
    Path              string // Relative path from plugins dir
    PluginURI         string // Optional
    Description       string // Optional
    Author            string // Optional
    AuthorURI         string // Optional
    RequiresAtLeast   string // WordPress version
    RequiresPHP       string // PHP version
    TextDomain        string // Optional
}

// ParseResult contains parsing results and any errors
type ParseResult struct {
    Plugin *PluginInfo
    Errors []error    // Non-fatal parsing errors
}
```

#### Parser Interface

```go
// Parser parses plugin headers from PHP file content
type Parser interface {
    // Parse extracts plugin information from file content
    // content: first 8KB of the plugin file
    Parse(content []byte) (*ParseResult, error)
}

type parser struct {
    // Compiled regex patterns
    headerPattern *regexp.Regexp
}

func NewParser() Parser {
    return &parser{
        headerPattern: compileHeaderPattern(),
    }
}
```

#### Parsing Algorithm

1. **Read first 8KB** (WordPress standard)
2. **Locate comment block** containing headers
3. **Extract each header field** using regex
4. **Validate required fields** (Plugin Name must exist)
5. **Return PluginInfo** or error

#### Regular Expression Pattern

```go
// Header field pattern matches:
// * Plugin Name: Example Plugin
// * Version: 1.2.3
// etc.

const headerFieldPattern = `(?m)^\s*\*\s*([A-Z][a-z\s]+):\s*(.+?)\s*$`

// Extracted fields:
// Group 1: Field name (e.g., "Plugin Name", "Version")
// Group 2: Field value (e.g., "Example Plugin", "1.2.3")
```

#### Test Cases

```go
func TestParser_Parse(t *testing.T) {
    tests := []struct {
        name        string
        content     string
        want        *PluginInfo
        wantErr     bool
    }{
        {
            name: "standard plugin header",
            content: `<?php
/**
 * Plugin Name: Test Plugin
 * Version: 1.2.3
 * Description: A test plugin
 */`,
            want: &PluginInfo{
                Name:    "Test Plugin",
                Version: "1.2.3",
                Description: "A test plugin",
            },
            wantErr: false,
        },
        {
            name: "minimal header (only Plugin Name)",
            content: `<?php
/*
Plugin Name: Minimal Plugin
*/`,
            want: &PluginInfo{
                Name:    "Minimal Plugin",
                Version: "", // No version
            },
            wantErr: false,
        },
        {
            name: "no header",
            content: `<?php
// Just a regular PHP file
class MyClass {}`,
            want:    nil,
            wantErr: true,
        },
        {
            name: "header beyond 8KB",
            content: strings.Repeat("// comment\n", 500) + `
/**
 * Plugin Name: Too Far
 */`,
            want:    nil,
            wantErr: true, // Should not find if beyond 8KB
        },
    }
    // ... test implementation
}
```

### Phase 2: Directory Scanner (TDD)

**Goal**: Scan plugin directory and find main plugin files

#### Scanner Interface

```go
// Scanner scans a WordPress plugins directory
type Scanner interface {
    // Scan finds all plugin files in the directory
    // pluginsDir: path to wp-content/plugins
    Scan(pluginsDir string) ([]string, error)
}

type scanner struct {
    maxDepth int // WordPress scans max 2 levels
}

func NewScanner() Scanner {
    return &scanner{
        maxDepth: 2,
    }
}
```

#### Scanning Algorithm

1. **Level 1**: Scan `plugins/*.php`
2. **Level 2**: Scan `plugins/*/*.php`
3. **Filter**: Only `.php` files
4. **Return**: List of file paths to check

#### Implementation

```go
func (s *scanner) Scan(pluginsDir string) ([]string, error) {
    var phpFiles []string

    // Level 1: Direct .php files in plugins directory
    level1Files, err := filepath.Glob(filepath.Join(pluginsDir, "*.php"))
    if err != nil {
        return nil, err
    }
    phpFiles = append(phpFiles, level1Files...)

    // Level 2: .php files in subdirectories
    dirs, err := os.ReadDir(pluginsDir)
    if err != nil {
        return nil, err
    }

    for _, dir := range dirs {
        if !dir.IsDir() {
            continue
        }

        level2Files, err := filepath.Glob(
            filepath.Join(pluginsDir, dir.Name(), "*.php"),
        )
        if err != nil {
            continue // Skip problematic directories
        }
        phpFiles = append(phpFiles, level2Files...)
    }

    return phpFiles, nil
}
```

#### Test Cases

```go
func TestScanner_Scan(t *testing.T) {
    tests := []struct {
        name      string
        structure map[string]string // path -> content
        wantFiles []string
        wantErr   bool
    }{
        {
            name: "single file plugin",
            structure: map[string]string{
                "hello-dolly/hello.php": "plugin content",
            },
            wantFiles: []string{"hello-dolly/hello.php"},
            wantErr:   false,
        },
        {
            name: "multi-file plugin",
            structure: map[string]string{
                "akismet/akismet.php":       "main file",
                "akismet/class.akismet.php": "class file",
            },
            wantFiles: []string{
                "akismet/akismet.php",
                "akismet/class.akismet.php",
            },
            wantErr: false,
        },
        {
            name: "ignore deep nesting",
            structure: map[string]string{
                "plugin/plugin.php":              "level 2",
                "plugin/includes/class.php":      "level 3 (ignore)",
            },
            wantFiles: []string{"plugin/plugin.php"},
            wantErr:   false,
        },
    }
    // ... test implementation with temp directories
}
```

### Phase 3: Main Detector (Integration)

**Goal**: Combine scanner and parser to detect all plugins

#### Detector Interface

```go
// Detector detects WordPress plugins
type Detector interface {
    // Detect scans a WordPress plugins directory and returns detected plugins
    Detect(pluginsDir string) ([]*PluginInfo, error)
}

type detector struct {
    scanner Scanner
    parser  Parser
}

func NewDetector() Detector {
    return &detector{
        scanner: NewScanner(),
        parser:  NewParser(),
    }
}
```

#### Detection Algorithm

```go
func (d *detector) Detect(pluginsDir string) ([]*PluginInfo, error) {
    // 1. Scan for PHP files
    phpFiles, err := d.scanner.Scan(pluginsDir)
    if err != nil {
        return nil, err
    }

    var plugins []*PluginInfo
    var errs []error

    // 2. Process each file
    for _, filePath := range phpFiles {
        // Read first 8KB
        content, err := readFirst8KB(filePath)
        if err != nil {
            errs = append(errs, err)
            continue
        }

        // Parse headers
        result, err := d.parser.Parse(content)
        if err != nil {
            // No valid header found, skip this file
            continue
        }

        // Set path and slug
        relPath, _ := filepath.Rel(pluginsDir, filePath)
        result.Plugin.Path = relPath
        result.Plugin.Slug = extractSlug(relPath)

        plugins = append(plugins, result.Plugin)
    }

    return plugins, nil
}

func readFirst8KB(filePath string) ([]byte, error) {
    f, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    // Read maximum 8192 bytes (8KB)
    buf := make([]byte, 8192)
    n, err := f.Read(buf)
    if err != nil && err != io.EOF {
        return nil, err
    }

    return buf[:n], nil
}

func extractSlug(relPath string) string {
    // Extract plugin slug from path
    // "akismet/akismet.php" -> "akismet"
    // "hello-dolly/hello.php" -> "hello-dolly"
    parts := strings.Split(relPath, string(os.PathSeparator))
    if len(parts) > 1 {
        return parts[0]
    }
    // Single file plugin: use filename without extension
    return strings.TrimSuffix(parts[0], ".php")
}
```

#### Integration Test

```go
func TestDetector_Detect(t *testing.T) {
    // Use real downloaded plugins in testdata/
    detector := NewDetector()

    plugins, err := detector.Detect("../../testdata/wp-content/plugins")
    if err != nil {
        t.Fatalf("Detect() error = %v", err)
    }

    // Should detect 100 plugins
    if len(plugins) != 100 {
        t.Errorf("Expected 100 plugins, got %d", len(plugins))
    }

    // Verify specific plugins
    pluginMap := make(map[string]*PluginInfo)
    for _, p := range plugins {
        pluginMap[p.Slug] = p
    }

    // Check Akismet
    if akismet, ok := pluginMap["akismet"]; ok {
        if akismet.Name != "Akismet Anti-spam: Spam Protection" {
            t.Errorf("Akismet name = %s", akismet.Name)
        }
        if akismet.Version != "5.5" {
            t.Errorf("Akismet version = %s", akismet.Version)
        }
    } else {
        t.Error("Akismet not detected")
    }

    // Check WooCommerce
    if wc, ok := pluginMap["woocommerce"]; ok {
        if wc.Version != "10.3.3" {
            t.Errorf("WooCommerce version = %s", wc.Version)
        }
    } else {
        t.Error("WooCommerce not detected")
    }
}
```

## Implementation Order (TDD)

### Step 1: Parser Tests & Implementation
```bash
# 1. Write parser tests
touch pkg/detector/parser_test.go

# 2. Run tests (should fail)
go test ./pkg/detector -v -run TestParser

# 3. Implement parser
touch pkg/detector/parser.go

# 4. Make tests pass
go test ./pkg/detector -v -run TestParser

# 5. Refactor
```

### Step 2: Scanner Tests & Implementation
```bash
# 1. Write scanner tests
touch pkg/detector/scanner_test.go

# 2. Run tests (should fail)
go test ./pkg/detector -v -run TestScanner

# 3. Implement scanner
touch pkg/detector/scanner.go

# 4. Make tests pass
go test ./pkg/detector -v -run TestScanner

# 5. Refactor
```

### Step 3: Detector Integration
```bash
# 1. Write detector tests
touch pkg/detector/detector_test.go

# 2. Implement detector
touch pkg/detector/detector.go

# 3. Run integration tests with real data
go test ./pkg/detector -v

# 4. Verify with 100 downloaded plugins
```

## Edge Cases to Handle

| Case | Handling |
|------|----------|
| Multiple headers in directory | Use first valid header found |
| No header in any .php file | Return empty, no error |
| Malformed header | Skip file, continue scanning |
| Permission denied | Log error, continue with other files |
| Symlinks | Follow symlinks (security check needed) |
| Non-UTF8 encoding | Try UTF-8, fall back to best effort |
| Header in wrong location | Won't detect (matches WordPress behavior) |

## Performance Considerations

- **Parallel scanning**: Process files concurrently
- **8KB read limit**: Don't load entire files
- **Regex compilation**: Compile once, reuse
- **Memory efficiency**: Stream processing, not loading all files

## Optional: Version Constant Detection

**Lower priority** - Can be added after main implementation

```go
// ParseWithConstant also extracts version from define() statement
func (p *parser) ParseWithConstant(content []byte) (*ParseResult, error) {
    result, err := p.Parse(content)
    if err != nil {
        return nil, err
    }

    // If version found in header, return it
    if result.Plugin.Version != "" {
        return result, nil
    }

    // Fallback: try to find version constant
    constantVersion := extractVersionConstant(content)
    if constantVersion != "" {
        result.Plugin.Version = constantVersion
    }

    return result, nil
}

func extractVersionConstant(content []byte) string {
    // Pattern: define('PLUGIN_VERSION', '1.2.3')
    pattern := regexp.MustCompile(`define\s*\(\s*['"](\w+_VERSION)['"]?\s*,\s*['"]([0-9.]+[^'"]*)['"]\s*\)`)
    matches := pattern.FindSubmatch(content)
    if len(matches) >= 3 {
        return string(matches[2])
    }
    return ""
}
```

## Success Criteria

- ✅ Detect all 100 downloaded plugins correctly
- ✅ Extract Plugin Name and Version for each
- ✅ Match WordPress `get_plugins()` behavior
- ✅ Pass all unit tests
- ✅ Pass integration tests with real plugins
- ✅ Performance: Scan 100 plugins in < 1 second

## Next Steps After Implementation

1. Create CLI tool (`cmd/detect-plugins`)
2. Add JSON output format
3. Integrate with WPScan API (Phase 4)
4. Create vulnerability scanner tool
