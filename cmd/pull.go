package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ioworker0/timage/pkg/config"
	"github.com/ioworker0/timage/pkg/proxy"
	"github.com/ioworker0/timage/pkg/registry"
	"github.com/ioworker0/timage/pkg/storage"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [image]",
	Short: "Pull an image from a registry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		imageRef := args[0]

		// Get proxy from flag
		proxyFlag, _ := cmd.Flags().GetString("proxy")

		// Parse image reference
		name, tag, registryURL := parseImageRef(imageRef)

		cmd.Printf("Pulling %s from %s...\n", imageRef, registryURL)

		// Get auth from config
		storageDir, err := config.GetConfigDir()
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		cfg, err := config.NewManager(storageDir)
		if err != nil {
			cmd.Printf("Error: Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Get proxy URL
		proxyURL := proxy.GetProxyURL(proxyFlag)
		if proxyURL == "" {
			proxyURL = cfg.GetRegistryProxy(registryURL)
		}

		// Get auth credentials
		username, password, _ := cfg.GetAuth(registryURL)
		auth := &registry.AuthConfig{
			Username: username,
			Password: password,
		}

		// Create registry client
		client, err := registry.NewClient(registryURL, auth, proxyURL)
		if err != nil {
			cmd.Printf("Error: Failed to create registry client: %v\n", err)
			os.Exit(1)
		}

		// Get manifest
		manifest, err := client.GetManifest(name, tag)
		if err != nil {
			cmd.Printf("Error: Failed to get manifest: %v\n", err)
			os.Exit(1)
		}

		cmd.Printf("Manifest: MediaType=%s, SchemaVersion=%d, Layers=%d, Manifests=%d\n",
			manifest.MediaType, manifest.SchemaVersion, len(manifest.Layers), len(manifest.Manifests))
		if manifest.Config.Digest != "" {
			cmd.Printf("Config digest: %s\n", manifest.Config.Digest)
		}

		// Create storage
		configDir, err := config.GetConfigDir()
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		store, err := storage.NewStore(configDir)
		if err != nil {
			cmd.Printf("Error: Failed to create store: %v\n", err)
			os.Exit(1)
		}

		// Download config blob
		cmd.Printf("Downloading config...\n")
		configPath, err := downloadBlobWithProgress(client, store, name, manifest.Config.Digest, "Config")
		if err != nil {
			cmd.Printf("Error: Failed to download config: %v\n", err)
			os.Exit(1)
		}

		// Save config
		configData, err := os.ReadFile(configPath)
		if err != nil {
			cmd.Printf("Error: Failed to read config: %v\n", err)
			os.Exit(1)
		}

		if err := store.SaveConfig(imageRef, configData); err != nil {
			cmd.Printf("Error: Failed to save config: %v\n", err)
			os.Exit(1)
		}

		// Download layers
		cmd.Printf("Downloading layers...\n")
		for i, layer := range manifest.Layers {
			layerName := fmt.Sprintf("Layer %d/%d", i+1, len(manifest.Layers))
			layerPath, err := downloadBlobWithProgress(client, store, name, layer.Digest, layerName)
			if err != nil {
				cmd.Printf("Error: Failed to download layer: %v\n", err)
				os.Exit(1)
			}

			// Save layer
			if err := store.SaveLayer(imageRef, layer.Digest, layerPath); err != nil {
				cmd.Printf("Error: Failed to save layer: %v\n", err)
				os.Exit(1)
			}
		}

		// Get raw manifest for storage (preserve exact format)
		// First fetch by tag to check if it's a manifest list
		manifestRaw, contentType, err := client.GetManifestRaw(name, tag)
		if err != nil {
			cmd.Printf("Error: Failed to get raw manifest: %v\n", err)
			os.Exit(1)
		}

		// Check if it's a manifest list
		isManifestList := contentType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
		                  contentType == "application/vnd.oci.image.index.v1+json"

		if isManifestList {
			// Parse to find amd64 manifest digest
			var index struct {
				Manifests []struct {
					Digest    string
					Platform  struct {
						Architecture string
						OS           string
					}
				}
			}
			if err := json.Unmarshal(manifestRaw, &index); err == nil {
				// Find amd64/linux manifest
				for _, m := range index.Manifests {
					if m.Platform.Architecture == "amd64" && m.Platform.OS == "linux" {
						// Fetch the actual platform-specific manifest
						manifestRaw, contentType, err = client.GetManifestRaw(name, m.Digest)
						if err != nil {
							cmd.Printf("Error: Failed to get platform manifest: %v\n", err)
							os.Exit(1)
						}
						break
					}
				}
			}
		}

		// Convert OCI format to Docker format if needed
		if contentType == "application/vnd.oci.image.manifest.v1+json" {
			// Replace OCI mediaTypes with Docker mediaTypes
			manifestRaw = bytes.ReplaceAll(manifestRaw,
				[]byte("application/vnd.oci.image.config.v1+json"),
				[]byte("application/vnd.docker.container.image.v1+json"))
			manifestRaw = bytes.ReplaceAll(manifestRaw,
				[]byte("application/vnd.oci.image.layer.v1.tar+gzip"),
				[]byte("application/vnd.docker.image.rootfs.diff.tar.gzip"))
			// Also update the top-level mediaType
			manifestRaw = bytes.ReplaceAll(manifestRaw,
				[]byte("application/vnd.oci.image.manifest.v1+json"),
				[]byte("application/vnd.docker.distribution.manifest.v2+json"))
		}

		// Save manifest
		if err := store.SaveManifestRaw(imageRef, manifestRaw); err != nil {
			cmd.Printf("Error: Failed to save manifest: %v\n", err)
			os.Exit(1)
		}

		cmd.Printf("\nSuccessfully pulled %s\n", imageRef)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

// parseImageRef parses an image reference into name, tag, and registry
func parseImageRef(imageRef string) (name, tag, registry string) {
	// Default values
	registry = "docker.io"
	tag = "latest"

	// Parse registry
	if idx := strings.Index(imageRef, "/"); idx != -1 {
		potentialRegistry := imageRef[:idx]
		if strings.Contains(potentialRegistry, ".") || strings.Contains(potentialRegistry, ":") {
			registry = potentialRegistry
			imageRef = imageRef[idx+1:]
		}
	}

	// Parse tag
	if idx := strings.Index(imageRef, ":"); idx != -1 {
		tag = imageRef[idx+1:]
		name = imageRef[:idx]
	} else {
		name = imageRef
	}

	// Add library/ prefix for official Docker Hub images
	if registry == "docker.io" && !strings.Contains(name, "/") {
		name = "library/" + name
	}

	return name, tag, registry
}

func downloadBlobWithProgress(client *registry.Client, store *storage.Store, name, digest, label string) (string, error) {
	// Validate digest
	if digest == "" {
		return "", fmt.Errorf("empty digest")
	}

	// For simplicity, download to temp file first
	tempPath := fmt.Sprintf("/tmp/timage-%s.tmp", digest[:12])

	// Create progress tracker
	progress := &progressTracker{
		label:  label,
		digest: digest,
		start:  time.Now(),
	}

	if err := client.DownloadBlobWithProgress(name, digest, tempPath, progress.update); err != nil {
		return "", err
	}

	// Print completion
	progress.finish()

	return tempPath, nil
}

// progressTracker tracks download progress
type progressTracker struct {
	label      string
	digest     string
	downloaded int64
	total      int64
	start      time.Time
	lastUpdate time.Time
}

// update updates the progress
func (p *progressTracker) update(downloaded, total int64) {
	p.downloaded = downloaded
	p.total = total
	now := time.Now()

	// Update at most every 100ms to avoid flickering
	if now.Sub(p.lastUpdate) < 100*time.Millisecond && downloaded < total {
		return
	}
	p.lastUpdate = now

	// Calculate progress
	percent := float64(downloaded) / float64(total) * 100

	// Build progress bar
	barWidth := 40
	filled := int(float64(barWidth) * percent / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Print progress
	fmt.Printf("\r[%s] %s %s",
		bar,
		p.label,
		formatBytes(total),
	)
}

// finish prints the completion message
func (p *progressTracker) finish() {
	// Build complete progress bar
	barWidth := 40
	bar := strings.Repeat("█", barWidth)

	fmt.Printf("\r[%s] %s %s\n",
		bar,
		p.label,
		formatBytes(p.total),
	)
}

// formatBytes formats a byte size
func formatBytes(b int64) string {
	if b >= 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(b)/(1024*1024))
	} else if b >= 1024 {
		return fmt.Sprintf("%.2f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%d B", b)
}

// formatDuration formats a duration
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
