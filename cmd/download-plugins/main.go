package main

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/masahiro331/go-wp-detector/pkg/wordpress"
)

const (
	defaultOutputDir = "testdata/wp-content/plugins"
)

type Config struct {
	Count     int
	OutputDir string
}

func main() {
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() Config {
	var cfg Config

	flag.IntVar(&cfg.Count, "count", 100, "Number of plugins to download")
	flag.StringVar(&cfg.OutputDir, "output", defaultOutputDir, "Output directory for plugins")
	flag.Parse()

	return cfg
}

func run(cfg Config) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	client := wordpress.NewClient()
	ctx := context.Background()

	log.Printf("Fetching top %d popular plugins from WordPress.org...", cfg.Count)

	// Calculate pagination
	const perPage = 100
	totalPages := (cfg.Count + perPage - 1) / perPage

	var allPlugins []wordpress.PluginInfo

	for page := 1; page <= totalPages; page++ {
		requestPerPage := perPage
		if page == totalPages {
			// Last page might need fewer plugins
			remaining := cfg.Count - len(allPlugins)
			if remaining < perPage {
				requestPerPage = remaining
			}
		}

		log.Printf("Fetching page %d/%d (per_page=%d)...", page, totalPages, requestPerPage)

		resp, err := client.QueryPlugins(ctx, "popular", requestPerPage, page)
		if err != nil {
			return fmt.Errorf("failed to query plugins: %w", err)
		}

		allPlugins = append(allPlugins, resp.Plugins...)

		if len(allPlugins) >= cfg.Count {
			allPlugins = allPlugins[:cfg.Count]
			break
		}

		// Rate limiting - be respectful to WordPress.org API
		time.Sleep(1 * time.Second)
	}

	log.Printf("Found %d plugins. Starting download...", len(allPlugins))

	// Download and extract plugins
	for i, plugin := range allPlugins {
		log.Printf("[%d/%d] Downloading %s (%s)...", i+1, len(allPlugins), plugin.Name, plugin.Version)

		if err := downloadAndExtractPlugin(ctx, client, plugin, cfg.OutputDir); err != nil {
			log.Printf("  ⚠️  Failed to download %s: %v", plugin.Slug, err)
			continue
		}

		log.Printf("  ✅ Successfully extracted to %s/%s", cfg.OutputDir, plugin.Slug)

		// Rate limiting
		if i < len(allPlugins)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	log.Printf("\n✅ Download complete! %d plugins saved to %s", len(allPlugins), cfg.OutputDir)

	return nil
}

func downloadAndExtractPlugin(ctx context.Context, client *wordpress.Client, plugin wordpress.PluginInfo, outputDir string) error {
	// Download plugin ZIP
	data, err := client.DownloadPlugin(ctx, plugin.DownloadLink)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract ZIP
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to read ZIP: %w", err)
	}

	// Extract all files
	for _, file := range zipReader.File {
		if err := extractFile(file, outputDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}

	return nil
}

func extractFile(file *zip.File, outputDir string) error {
	// Prevent path traversal attacks
	filePath := filepath.Join(outputDir, file.Name)
	if !filepath.HasPrefix(filePath, filepath.Clean(outputDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(filePath, file.Mode())
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	// Extract file
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, rc)
	return err
}
