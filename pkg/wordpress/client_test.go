package wordpress_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/masahiro331/go-wp-detector/pkg/wordpress"
)

func TestClient_QueryPlugins(t *testing.T) {
	tests := []struct {
		name           string
		browse         string
		perPage        int
		page           int
		serverResponse wordpress.QueryPluginsResponse
		wantErr        bool
	}{
		{
			name:    "query popular plugins successfully",
			browse:  "popular",
			perPage: 5,
			page:    1,
			serverResponse: wordpress.QueryPluginsResponse{
				Info: wordpress.QueryInfo{
					Page:    1,
					Pages:   100,
					Results: 500,
				},
				Plugins: []wordpress.PluginInfo{
					{
						Name:           "Akismet Anti-Spam",
						Slug:           "akismet",
						Version:        "5.0",
						DownloadLink:   "https://downloads.wordpress.org/plugin/akismet.5.0.zip",
						ActiveInstalls: 5000000,
						Downloaded:     100000000,
						Rating:         95.0,
					},
					{
						Name:           "Jetpack",
						Slug:           "jetpack",
						Version:        "12.0",
						DownloadLink:   "https://downloads.wordpress.org/plugin/jetpack.12.0.zip",
						ActiveInstalls: 4000000,
						Downloaded:     90000000,
						Rating:         92.0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "query with zero per_page should fail",
			browse:  "popular",
			perPage: 0,
			page:    1,
			wantErr: true,
		},
		{
			name:    "query with negative page should fail",
			browse:  "popular",
			perPage: 10,
			page:    -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantErr && (tt.perPage <= 0 || tt.page < 1) {
					// Client should validate before making request
					t.Error("Server should not be called for invalid parameters")
					return
				}

				// Verify request parameters
				query := r.URL.Query()
				if query.Get("action") != "query_plugins" {
					t.Errorf("Expected action=query_plugins, got %s", query.Get("action"))
				}

				// Return mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := wordpress.NewClient(wordpress.WithBaseURL(server.URL))

			ctx := context.Background()
			resp, err := client.QueryPlugins(ctx, tt.browse, tt.perPage, tt.page)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryPlugins() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Fatal("Expected non-nil response")
				}
				if len(resp.Plugins) != len(tt.serverResponse.Plugins) {
					t.Errorf("Expected %d plugins, got %d", len(tt.serverResponse.Plugins), len(resp.Plugins))
				}
				if len(resp.Plugins) > 0 {
					if resp.Plugins[0].Slug != tt.serverResponse.Plugins[0].Slug {
						t.Errorf("Expected first plugin slug %s, got %s",
							tt.serverResponse.Plugins[0].Slug, resp.Plugins[0].Slug)
					}
				}
			}
		})
	}
}

func TestClient_GetPluginInfo(t *testing.T) {
	tests := []struct {
		name           string
		slug           string
		serverResponse wordpress.PluginInfo
		wantErr        bool
	}{
		{
			name: "get plugin info successfully",
			slug: "akismet",
			serverResponse: wordpress.PluginInfo{
				Name:         "Akismet Anti-Spam",
				Slug:         "akismet",
				Version:      "5.0",
				DownloadLink: "https://downloads.wordpress.org/plugin/akismet.5.0.zip",
			},
			wantErr: false,
		},
		{
			name:    "empty slug should fail",
			slug:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantErr && tt.slug == "" {
					t.Error("Server should not be called for invalid parameters")
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := wordpress.NewClient(wordpress.WithBaseURL(server.URL))

			ctx := context.Background()
			info, err := client.GetPluginInfo(ctx, tt.slug)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPluginInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if info == nil {
					t.Fatal("Expected non-nil plugin info")
				}
				if info.Slug != tt.serverResponse.Slug {
					t.Errorf("Expected slug %s, got %s", tt.serverResponse.Slug, info.Slug)
				}
			}
		})
	}
}

func TestClient_DownloadPlugin(t *testing.T) {
	tests := []struct {
		name        string
		downloadURL string
		wantErr     bool
	}{
		{
			name:        "download plugin successfully",
			downloadURL: "/plugin.zip",
			wantErr:     false,
		},
		{
			name:        "empty download URL should fail",
			downloadURL: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantErr && tt.downloadURL == "" {
					t.Error("Server should not be called for invalid parameters")
					return
				}

				// Return mock ZIP content
				w.Header().Set("Content-Type", "application/zip")
				w.Write([]byte("PK\x03\x04")) // ZIP file header
			}))
			defer server.Close()

			client := wordpress.NewClient()

			downloadURL := tt.downloadURL
			if downloadURL != "" {
				downloadURL = server.URL + downloadURL
			}

			ctx := context.Background()
			data, err := client.DownloadPlugin(ctx, downloadURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadPlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(data) == 0 {
					t.Error("Expected non-empty plugin data")
				}
			}
		})
	}
}
