# WPScan API Investigation and Usage Plan

## Overview

WPScan provides a WordPress Vulnerability Database API for querying known vulnerabilities in WordPress core, plugins, and themes.

## API Details

### Base Information
- **Base URL:** `https://wpscan.com/api/v3/`
- **Format:** JSON (OpenAPI 3.0 specification)
- **Authentication:** Token-based via HTTP header
- **Official Docs:** https://wpscan.com/docs/api/v3/

### Authentication

**Header Format:**
```
Authorization: Token token=YOUR_API_TOKEN
```

**Getting API Token:**
1. Register an account at https://wpscan.com/
2. Get API token from profile page: https://wpscan.com/profile
3. Free tier provides 25 requests per day

**Environment Variable:**
The API token can be loaded from environment variable:
```bash
export WPSCAN_API_TOKEN=your_token_here
```

## API Endpoints

### 1. Plugin Vulnerabilities

**Endpoint:** `GET /plugins/{slug}`

Query all known vulnerabilities for a specific plugin.

**Parameters:**
- `slug` (required): WordPress plugin slug (e.g., "akismet")

**Example Request:**
```bash
curl -H "Authorization: Token token=YOUR_TOKEN" \
  https://wpscan.com/api/v3/plugins/akismet
```

**Response Structure:**
```json
{
  "slug": "akismet",
  "vulnerabilities": [
    {
      "id": "vulnerability-id",
      "title": "Vulnerability Title",
      "created_at": "2023-01-01T00:00:00.000Z",
      "updated_at": "2023-01-02T00:00:00.000Z",
      "published_date": "2023-01-01",
      "vuln_type": "XSS/SQL Injection/etc",
      "references": {
        "url": ["https://..."],
        "cve": ["CVE-2023-xxxxx"]
      },
      "fixed_in": "1.2.3"
    }
  ]
}
```

### 2. Version-Specific Plugin Vulnerabilities

**Endpoint:** `GET /plugins/{slug}/{version}`

Query vulnerabilities that affect a specific plugin version.

**Parameters:**
- `slug` (required): WordPress plugin slug
- `version` (required): Plugin version (e.g., "1.2.3")

**Example Request:**
```bash
curl -H "Authorization: Token token=YOUR_TOKEN" \
  https://wpscan.com/api/v3/plugins/akismet/1.2.3
```

### 3. WordPress Core Vulnerabilities

**Endpoint:** `GET /wordpresses/{version}`

Query vulnerabilities for specific WordPress version.

### 4. Theme Vulnerabilities

**Endpoint:** `GET /themes/{slug}`

Query vulnerabilities for specific theme.

### 5. User Status

**Endpoint:** `GET /status`

Check API usage and remaining requests.

**Example Request:**
```bash
curl -H "Authorization: Token token=YOUR_TOKEN" \
  https://wpscan.com/api/v3/status
```

## Rate Limits

### Free Tier
- **Requests:** 25 per day
- **Cost:** Free
- **Suitable for:** Small-scale scanning, personal projects

### Paid Tiers
- Higher request limits available
- Enterprise options for bulk scanning
- Contact WPScan for pricing

## Implementation Plan

### Phase 1: API Client Design

**Package Structure:**
```go
package wpscan

type Client struct {
    baseURL    string
    apiToken   string
    httpClient *http.Client
}

type ClientOption func(*Client)

// NewClient creates a new WPScan API client
func NewClient(apiToken string, opts ...ClientOption) *Client

// GetPluginVulnerabilities queries all vulnerabilities for a plugin
func (c *Client) GetPluginVulnerabilities(ctx context.Context, slug string) (*VulnerabilityReport, error)

// GetPluginVersionVulnerabilities queries vulnerabilities for specific plugin version
func (c *Client) GetPluginVersionVulnerabilities(ctx context.Context, slug, version string) (*VulnerabilityReport, error)

// GetStatus checks API usage status
func (c *Client) GetStatus(ctx context.Context) (*Status, error)
```

### Phase 2: Data Models

```go
type VulnerabilityReport struct {
    Slug            string          `json:"slug"`
    Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

type Vulnerability struct {
    ID            string            `json:"id"`
    Title         string            `json:"title"`
    CreatedAt     time.Time         `json:"created_at"`
    UpdatedAt     time.Time         `json:"updated_at"`
    PublishedDate string            `json:"published_date"`
    VulnType      string            `json:"vuln_type"`
    References    VulnReferences    `json:"references"`
    FixedIn       string            `json:"fixed_in"`
}

type VulnReferences struct {
    URL []string `json:"url"`
    CVE []string `json:"cve"`
}

type Status struct {
    RequestsRemaining int       `json:"requests_remaining"`
    RequestsLimit     int       `json:"requests_limit"`
    PlanName          string    `json:"plan"`
    // ... other fields
}
```

### Phase 3: Testing Strategy

**Unit Tests:**
- Mock HTTP responses for API calls
- Test error handling (rate limits, authentication errors)
- Test JSON parsing

**Integration Tests:**
- Use test API token (if available)
- Test with known vulnerable plugins
- Verify rate limiting behavior

**Test Data:**
```go
func TestGetPluginVulnerabilities(t *testing.T) {
    tests := []struct {
        name    string
        slug    string
        wantErr bool
    }{
        {"known vulnerable plugin", "old-plugin", false},
        {"non-existent plugin", "nonexistent-plugin-xyz", false},
        {"empty slug", "", true},
    }
    // ... table-driven test implementation
}
```

### Phase 4: Rate Limiting Handling

**Considerations:**
- Free tier: 25 requests/day
- Need to implement rate limiting awareness
- Cache results to minimize API calls
- Handle rate limit errors gracefully

**Implementation:**
```go
type RateLimiter struct {
    requestsRemaining int
    resetTime         time.Time
}

// Check if we can make a request
func (r *RateLimiter) CanRequest() bool

// Update rate limit info from response headers
func (r *RateLimiter) UpdateFromResponse(resp *http.Response)
```

## Security Considerations

### API Token Storage
- **Environment Variable:** Preferred method
- **Config File:** Ensure proper file permissions (0600)
- **Never commit:** Add to .gitignore
- **CI/CD:** Use secrets management

### Error Handling
- Don't expose API token in error messages
- Log errors without sensitive data
- Handle network failures gracefully

## Usage Examples

### Basic Scan
```go
client := wpscan.NewClient(os.Getenv("WPSCAN_API_TOKEN"))

report, err := client.GetPluginVulnerabilities(ctx, "akismet")
if err != nil {
    log.Fatal(err)
}

for _, vuln := range report.Vulnerabilities {
    fmt.Printf("Vulnerability: %s\n", vuln.Title)
    fmt.Printf("Fixed in: %s\n", vuln.FixedIn)
}
```

### Version-Specific Scan
```go
report, err := client.GetPluginVersionVulnerabilities(ctx, "akismet", "4.0.0")
if err != nil {
    log.Fatal(err)
}

if len(report.Vulnerabilities) > 0 {
    fmt.Println("⚠️  Vulnerabilities found!")
} else {
    fmt.Println("✅ No known vulnerabilities")
}
```

## Next Steps

1. ✅ Document API endpoints and authentication
2. ✅ Design client interface
3. ⏳ Obtain API token from WPScan
4. ⏳ Implement basic client with authentication
5. ⏳ Add vulnerability query methods
6. ⏳ Implement rate limiting
7. ⏳ Write unit tests with mocked responses
8. ⏳ Write integration tests (if API token available)
9. ⏳ Integrate with detector package

## References

- WPScan API Documentation: https://wpscan.com/docs/api/v3/
- WPScan API Main Page: https://wpscan.com/api/
- WPScan GitHub: https://github.com/wpscanteam/wpscan
