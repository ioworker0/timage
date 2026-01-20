package cmd

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/ioworker0/timage/pkg/config"
	"github.com/ioworker0/timage/pkg/proxy"
	"github.com/ioworker0/timage/pkg/registry"
	"github.com/ioworker0/timage/pkg/storage"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [image]",
	Short: "Push an image to a registry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		imageRef := args[0]

		// Get proxy from flag
		proxyFlag, _ := cmd.Flags().GetString("proxy")

		// Parse image reference
		name, tag, registryURL := parseImageRef(imageRef)

		cmd.Printf("Pushing %s to %s...\n", imageRef, registryURL)

		// Get storage
		storageDir, err := config.GetStorageDir()
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		store, err := storage.NewStore(storageDir)
		if err != nil {
			cmd.Printf("Error: Failed to create store: %v\n", err)
			os.Exit(1)
		}

		// Check if image exists locally
		if !store.ImageExists(imageRef) {
			cmd.Printf("Error: Image '%s' not found locally\n", imageRef)
			os.Exit(1)
		}

		// Load manifest
		manifest, err := store.LoadManifest(imageRef)
		if err != nil {
			cmd.Printf("Error: Failed to load manifest: %v\n", err)
			os.Exit(1)
		}

		// Get auth from config
		configDir, err := config.GetConfigDir()
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		cfg, err := config.NewManager(configDir)
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

		// Upload config blob
		cmd.Printf("Uploading config...\n")
		configPath := store.GetConfigPath(imageRef)

		// Try to upload config blob (Harbor will check if it exists)
		err = client.UploadBlob(name, manifest.Config.Digest, configPath)
		if err != nil {
			// Check if it's a "blob already exists" error
			if strings.Contains(err.Error(), "BLOB_UPLOAD_INVALID") ||
				strings.Contains(err.Error(), "already exists") ||
				strings.Contains(err.Error(), "exist blob") {
				cmd.Printf("  Config already exists, skipping\n")
			} else {
				cmd.Printf("Error: Failed to upload config: %v\n", err)
				os.Exit(1)
			}
		}

		// Upload layers
		cmd.Printf("Uploading %d layers...\n", len(manifest.Layers))
		for i, layer := range manifest.Layers {
			layerPath := store.GetLayerPath(imageRef, layer.Digest)

			cmd.Printf("  [%d/%d] %s\n", i+1, len(manifest.Layers), layer.Digest[:12])

			// Try to upload layer blob (Harbor will check if it exists)
			err = client.UploadBlob(name, layer.Digest, layerPath)
			if err != nil {
				// Check if it's a "blob already exists" error
				if strings.Contains(err.Error(), "BLOB_UPLOAD_INVALID") ||
					strings.Contains(err.Error(), "already exists") ||
					strings.Contains(err.Error(), "exist blob") {
					cmd.Printf("    Already exists, skipping\n")
				} else {
					cmd.Printf("Error: Failed to upload layer: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// Upload manifest
		cmd.Printf("Uploading manifest...\n")

		// Load raw manifest (preserve exact format from pull)
		manifestData, err := store.LoadManifestRaw(imageRef)
		if err != nil {
			cmd.Printf("Error: Failed to load manifest: %v\n", err)
			os.Exit(1)
		}

		// Parse and clean the manifest
		var manifestObj map[string]interface{}
		if err := json.Unmarshal(manifestData, &manifestObj); err != nil {
			cmd.Printf("Error: Failed to parse manifest: %v\n", err)
			os.Exit(1)
		}

		// Get content type
		contentType, _ := manifestObj["mediaType"].(string)

		// Check if it's a manifest list/index
		if contentType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
			contentType == "application/vnd.oci.image.index.v1+json" {
			cmd.Printf("Error: Cannot push manifest list/index. Please pull and push individual platform manifests.\n")
			os.Exit(1)
		}

		// Remove 'manifests' field if it exists (even if null) to ensure it's a pure manifest
		// Some registries reject manifests with a 'manifests' field during docker build
		delete(manifestObj, "manifests")

		// Ensure it's a proper manifest v2
		if contentType == "" {
			manifestObj["mediaType"] = "application/vnd.docker.distribution.manifest.v2+json"
			contentType = "application/vnd.docker.distribution.manifest.v2+json"
		}

		// Re-marshal the cleaned manifest
		manifestData, err = json.Marshal(manifestObj)
		if err != nil {
			cmd.Printf("Error: Failed to marshal manifest: %v\n", err)
			os.Exit(1)
		}

		// Upload manifest
		if err := client.PutManifest(name, tag, manifestData, contentType); err != nil {
			cmd.Printf("Error: Failed to upload manifest: %v\n", err)
			os.Exit(1)
		}

		cmd.Printf("\nSuccessfully pushed %s\n", imageRef)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
