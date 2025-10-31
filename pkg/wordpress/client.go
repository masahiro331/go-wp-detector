package wordpress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	defaultBaseURL = "https://api.wordpress.org/plugins/info/1.2/"
)

// Client is a WordPress.org API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ClientOption is a functional option for Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the client (mainly for testing)
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new WordPress.org API client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// QueryInfo contains pagination information for query results
type QueryInfo struct {
	Page    int `json:"page"`
	Pages   int `json:"pages"`
	Results int `json:"results"`
}

// PluginInfo contains detailed information about a WordPress plugin
type PluginInfo struct {
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Version        string  `json:"version"`
	DownloadLink   string  `json:"download_link"`
	ActiveInstalls int     `json:"active_installs"`
	Downloaded     int     `json:"downloaded"`
	Rating         float64 `json:"rating"`
	NumRatings     int     `json:"num_ratings"`
	Homepage       string  `json:"homepage"`
	ShortDesc      string  `json:"short_description"`
	Requires       string  `json:"requires"`
	Tested         string  `json:"tested"`
	RequiresPHP    string  `json:"requires_php"`
}

// QueryPluginsResponse is the response from the query_plugins API
type QueryPluginsResponse struct {
	Info    QueryInfo    `json:"info"`
	Plugins []PluginInfo `json:"plugins"`
}

// QueryPlugins queries WordPress plugins from the WordPress.org API
// browse: "popular", "featured", "updated", "new"
// perPage: number of results per page
// page: page number (1-based)
func (c *Client) QueryPlugins(ctx context.Context, browse string, perPage, page int) (*QueryPluginsResponse, error) {
	if perPage <= 0 {
		return nil, fmt.Errorf("perPage must be greater than 0")
	}
	if page < 1 {
		return nil, fmt.Errorf("page must be 1 or greater")
	}

	params := url.Values{}
	params.Set("action", "query_plugins")
	params.Set("request[browse]", browse)
	params.Set("request[per_page]", fmt.Sprintf("%d", perPage))
	params.Set("request[page]", fmt.Sprintf("%d", page))

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result QueryPluginsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetPluginInfo retrieves detailed information about a specific plugin
func (c *Client) GetPluginInfo(ctx context.Context, slug string) (*PluginInfo, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug cannot be empty")
	}

	params := url.Values{}
	params.Set("action", "plugin_information")
	params.Set("request[slug]", slug)

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result PluginInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DownloadPlugin downloads a plugin ZIP file from the given URL
func (c *Client) DownloadPlugin(ctx context.Context, downloadURL string) ([]byte, error) {
	if downloadURL == "" {
		return nil, fmt.Errorf("download URL cannot be empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}
