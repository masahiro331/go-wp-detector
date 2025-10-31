# WordPress Plugin Specification and Detection

This document describes the WordPress plugin specification based on official documentation and WordPress core code analysis.

## Table of Contents

1. [Plugin Directory Structure](#plugin-directory-structure)
2. [Plugin Header Specification](#plugin-header-specification)
3. [How WordPress Detects Plugins](#how-wordpress-detects-plugins)
4. [Detection Implementation Strategy](#detection-implementation-strategy)

## Plugin Directory Structure

### Standard Structure

WordPress plugins follow this recommended structure:

```
/plugin-name
  plugin-name.php          # Main plugin file (required)
  uninstall.php           # Optional cleanup handler
  readme.txt              # Plugin documentation
  /languages              # Translation files
  /includes               # Core plugin logic
  /admin                  # Admin-specific files
    /js
    /css
    /images
  /public                 # Frontend files
    /js
    /css
    /images
```

### Directory Depth

**Important**: WordPress only scans **two levels maximum**:
- **Level 1**: Files directly in `wp-content/plugins/` (e.g., `hello-dolly/hello.php`)
- **Level 2**: Files within immediate subdirectories (e.g., `akismet/akismet.php`)

Plugins nested deeper than two levels are not detected by WordPress.

### Main Plugin File

- Only **one file** in the plugin directory should contain the plugin header
- The main file typically shares the same name as the plugin directory (e.g., `akismet/akismet.php`)
- However, **this is convention, not requirement** - WordPress detects plugins by header presence, not filename

## Plugin Header Specification

### Required Header Fields

Only **one field is required**:

| Field | Description |
|-------|-------------|
| **Plugin Name** | The name displayed in WordPress admin (required) |

### Standard Optional Headers

| Field | Description | Example |
|-------|-------------|---------|
| **Plugin URI** | Plugin's homepage (must be unique URL) | `https://example.com/my-plugin` |
| **Description** | Short description (< 140 chars, no newlines) | `A simple plugin to...` |
| **Version** | Current version number | `1.2.3` |
| **Requires at least** | Minimum WordPress version | `6.0` |
| **Requires PHP** | Minimum PHP version | `7.4` |
| **Author** | Plugin creator(s), comma-separated | `John Doe, Jane Smith` |
| **Author URI** | Author's website | `https://example.com` |
| **License** | License slug | `GPLv2` |
| **License URI** | Full license URL | `https://www.gnu.org/licenses/gpl-2.0.html` |
| **Text Domain** | Translation identifier | `my-plugin` |
| **Domain Path** | Translation files directory | `/languages` |
| **Network** | Network activation only | `true` |
| **Update URI** | Prevents WordPress.org conflicts | Custom update URL |
| **Requires Plugins** | Plugin dependencies | `woocommerce, jetpack` |

### Header Format

Headers are PHP comments at the beginning of the main plugin file:

```php
<?php
/**
 * Plugin Name: Example Plugin
 * Plugin URI: https://example.com/my-plugin
 * Description: This is a short description of what the plugin does.
 * Version: 1.2.3
 * Requires at least: 6.0
 * Requires PHP: 7.4
 * Author: John Doe
 * Author URI: https://example.com
 * License: GPLv2 or later
 * License URI: https://www.gnu.org/licenses/gpl-2.0.html
 * Text Domain: example-plugin
 * Domain Path: /languages
 */
```

### Header Formatting Rules

1. **Location**: Headers must be within the **first 8KB** of the file
2. **Line format**: Each header must be on its own line
3. **No newlines**: Description cannot contain newlines
4. **Comment style**: Standard PHP block comment (`/** */` or `/* */`)
5. **Format**: `Field Name: Value`

### Version Comparison

The `Version` field uses PHP's `version_compare()` function for comparisons, supporting formats like:
- `1.0`
- `1.0.3`
- `2.0-beta1`

## How WordPress Detects Plugins

### Detection Algorithm

WordPress uses the `get_plugins()` function (`wp-admin/includes/plugin.php`):

1. **Open plugins directory**: `wp-content/plugins/`
2. **Scan Level 1**: Find all `.php` files in root
3. **Scan Level 2**: For each subdirectory, find `.php` files inside
4. **Parse headers**: Read first 8KB of each `.php` file with `get_plugin_data()`
5. **Validate**: Files with valid `Plugin Name` header are recognized as plugins
6. **Cache results**: Results are cached for performance

### Pseudo-code

```
function get_plugins():
    plugins = []

    # Level 1: Root .php files
    for file in plugins_dir/*.php:
        if has_plugin_header(file):
            plugins.add(file)

    # Level 2: Subdirectory .php files
    for subdir in plugins_dir/*/:
        for file in subdir/*.php:
            if has_plugin_header(file):
                plugins.add(subdir/file)

    return plugins

function has_plugin_header(file):
    content = read_first_8kb(file)
    return "Plugin Name:" in content
```

### Key Characteristics

- **Header-based detection**: Filename doesn't matter; only the header presence
- **8KB limit**: Headers must be in the first 8KB (8192 bytes)
- **Two-level scan**: Maximum depth of 2 directories
- **Single header**: Only one file per plugin should have headers
- **Case-sensitive**: Header field names are case-sensitive

## Detection Implementation Strategy

### For go-wp-detector

Based on WordPress's detection algorithm, our detector should:

#### 1. Directory Scanning

```go
// Scan pattern:
// - plugins/*/*.php (Level 1)
// - plugins/*/*/*.php (Level 2)
```

#### 2. File Parsing

```go
// For each .php file:
// 1. Read first 8KB
// 2. Search for plugin header comment
// 3. Extract header fields using regex
// 4. Validate "Plugin Name" exists
```

#### 3. Header Parsing Strategy

**Regular Expression Pattern**:
```regex
(?:^|[\r\n]+)\s*\*\s*([A-Z][a-z\s]+):\s*(.+)
```

This pattern matches:
- Line start or newline
- Optional whitespace + `*` (comment marker)
- Field name (capitalized words with spaces)
- Colon separator
- Field value

**Header Fields to Extract** (in priority order):

1. **Plugin Name** (required for detection)
2. **Version** (critical for vulnerability scanning)
3. **Requires at least** (WordPress compatibility)
4. **Requires PHP** (PHP compatibility)
5. **Description** (optional, for information)
6. **Author** (optional, for information)

#### 4. Edge Cases to Handle

| Case | Handling Strategy |
|------|------------------|
| Multiple headers in directory | Use first valid header found |
| No header in .php files | Skip directory |
| Malformed header | Extract what's possible, mark as incomplete |
| Non-UTF8 encoding | Use fallback encoding detection |
| Large files | Only read first 8KB |
| Nested deeper than 2 levels | Ignore (not WordPress-compliant) |

#### 5. Performance Considerations

- **Read limit**: Maximum 8KB per file
- **File filtering**: Skip non-.php files early
- **Caching**: Cache results for repeated scans
- **Concurrency**: Process multiple plugins in parallel

### Example Detection Flow

```
Input: /var/www/wp-content/plugins/

Step 1: Find PHP files
  ├── hello-dolly/hello.php        ✓ Found
  ├── akismet/akismet.php          ✓ Found
  ├── akismet/class.akismet.php    ✓ Found (but no header)
  └── woocommerce/woocommerce.php  ✓ Found

Step 2: Read first 8KB of each file

Step 3: Parse headers
  ├── hello-dolly/hello.php
  │   ├── Plugin Name: Hello Dolly
  │   └── Version: 1.7.2
  ├── akismet/akismet.php
  │   ├── Plugin Name: Akismet Anti-spam
  │   └── Version: 5.5
  ├── akismet/class.akismet.php    ✗ No header (skip)
  └── woocommerce/woocommerce.php
      ├── Plugin Name: WooCommerce
      └── Version: 10.3.3

Step 4: Return detected plugins
  [
    {name: "Hello Dolly", version: "1.7.2", path: "hello-dolly/hello.php"},
    {name: "Akismet Anti-spam", version: "5.5", path: "akismet/akismet.php"},
    {name: "WooCommerce", version: "10.3.3", path: "woocommerce/woocommerce.php"}
  ]
```

## Real-World Examples

### Example 1: Single-file Plugin (Hello Dolly)

```
Structure:
  hello-dolly/
    hello.php         # Contains header
    readme.txt

Detection:
  - Scan: hello-dolly/hello.php
  - Header found: Plugin Name: Hello Dolly
  - Result: hello-dolly/hello.php
```

### Example 2: Multi-file Plugin (Akismet)

```
Structure:
  akismet/
    akismet.php                      # Contains header
    class.akismet.php                # No header
    class.akismet-admin.php          # No header
    _inc/                            # Assets
    views/                           # Templates

Detection:
  - Scan: akismet/akismet.php         ✓ Header found
  - Scan: akismet/class.akismet.php   ✗ No header (skip)
  - Result: akismet/akismet.php
```

### Example 3: Complex Plugin (WooCommerce)

```
Structure:
  woocommerce/
    woocommerce.php                  # Main file with header
    includes/
      class-woocommerce.php          # Core classes
      wc-*.php                       # Helper functions
    templates/                       # Template files
    assets/                          # CSS/JS

Detection:
  - Scan: woocommerce/woocommerce.php ✓ Header found
  - Scan: woocommerce/includes/*      ✗ Depth > 2 (not scanned)
  - Result: woocommerce/woocommerce.php
```

## Implementation Checklist

- [ ] Scan wp-content/plugins with max depth 2
- [ ] Filter .php files only
- [ ] Read first 8KB of each file
- [ ] Parse plugin header comments
- [ ] Extract Plugin Name (required)
- [ ] Extract Version (critical for scanning)
- [ ] Extract other optional fields
- [ ] Handle malformed headers gracefully
- [ ] Return structured plugin metadata
- [ ] Cache results for performance

## Version Information Beyond Headers

### Common Pattern: Version Constants

While the `Version` header in the plugin file is the **primary and official source**, many plugins also define version constants in PHP for programmatic access. This is a **best practice but not a requirement**.

### Why Plugins Define Version Constants

1. **Programmatic Access**: Code can access version without parsing file headers
2. **Performance**: Faster than calling `get_plugin_data()` repeatedly
3. **Consistency**: Single source of truth for version-dependent code
4. **Database Migrations**: Track installed version vs current version

### Real-World Examples

#### Contact Form 7
```php
/**
 * Version: 6.1.3
 */
define( 'WPCF7_VERSION', '6.1.3' );  // Line 15
```

#### Elementor
```php
/**
 * Version: 3.32.5
 */
define( 'ELEMENTOR_VERSION', '3.32.5' );  // Line 31
```

#### Jetpack
```php
/**
 * Version: 15.1.1
 */
if ( ! defined( 'JETPACK__VERSION' ) ) {
    define( 'JETPACK__VERSION', '15.1.1' );  // Line 41
}
```

#### Wordfence
```php
/**
 * Version: 8.1.0
 */
define('WORDFENCE_VERSION', '8.1.0');  // Line 41
```

### Naming Conventions for Version Constants

Common patterns observed:

| Pattern | Examples | Frequency |
|---------|----------|-----------|
| `PLUGINNAME_VERSION` | `WPCF7_VERSION`, `ELEMENTOR_VERSION` | Most common |
| `PLUGINNAME__VERSION` | `JETPACK__VERSION` | Double underscore variant |
| `PLUGINNAME_VER` | Rare | Less common |
| Mixed case | `WooCommerceVersion` | Very rare |

### Location of Version Constants

Version constants are typically defined:
- **Immediately after** the plugin header comment (lines 15-50)
- **Before** any includes or class definitions
- **Within the first 100 lines** of the main plugin file

### Detection Strategy

For the detector implementation:

1. **Primary Source**: Always extract version from plugin header
   - Header `Version:` field is the official WordPress specification
   - This is what WordPress uses internally

2. **Fallback/Validation**: Optionally check for version constant
   - If header version is missing or malformed
   - For cross-validation to detect discrepancies
   - Useful for detecting potential bugs in plugin code

3. **Detection Pattern** for version constants:
```regex
define\s*\(\s*['"](\w+_VERSION|VERSION)['"]?\s*,\s*['"]([0-9.]+(?:-[a-zA-Z0-9]+)?)['"]\s*\)
```

### Important Notes

- **Header is authoritative**: WordPress only reads the header version
- **Constants may differ**: Some plugins have mismatched header/constant versions (bugs)
- **Not all plugins use constants**: Smaller plugins may skip this pattern
- **Some use class properties**: Object-oriented plugins may use `$this->version` instead

### Recommendation for Detector

**For go-wp-detector implementation:**

1. ✅ **Required**: Extract version from plugin header
2. ⚠️ **Optional**: Extract version from constant for validation
3. ⚠️ **Warn on mismatch**: If header and constant versions differ
4. ❌ **Don't rely solely on constants**: Some plugins don't define them

## References

- [WordPress Plugin Handbook - Header Requirements](https://developer.wordpress.org/plugins/plugin-basics/header-requirements/)
- [WordPress Plugin Handbook - Best Practices](https://developer.wordpress.org/plugins/plugin-basics/best-practices/)
- [WordPress Core - get_plugins()](https://developer.wordpress.org/reference/functions/get_plugins/)
- [WordPress Core - get_plugin_data()](https://developer.wordpress.org/reference/functions/get_plugin_data/)
- [WordPress Core - plugin.php Source](https://github.com/WordPress/WordPress/blob/master/wp-admin/includes/plugin.php)
