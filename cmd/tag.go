package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ioworker0/timage/pkg/config"
	"github.com/ioworker0/timage/pkg/storage"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag [source] [target]",
	Short: "Tag an image",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		target := args[1]

		// Get storage directory
		storageDir, err := config.GetStorageDir()
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Create store
		store, err := storage.NewStore(storageDir)
		if err != nil {
			cmd.Printf("Error: Failed to create store: %v\n", err)
			os.Exit(1)
		}

		// Check if source exists
		if !store.ImageExists(source) {
			cmd.Printf("Error: Source image '%s' not found\n", source)
			os.Exit(1)
		}

		// Copy manifest
		manifest, err := store.LoadManifest(source)
		if err != nil {
			cmd.Printf("Error: Failed to load manifest: %v\n", err)
			os.Exit(1)
		}

		if err := store.SaveManifest(target, manifest); err != nil {
			cmd.Printf("Error: Failed to save manifest: %v\n", err)
			os.Exit(1)
		}

		// Copy config
		configData, err := store.LoadConfig(source)
		if err != nil {
			cmd.Printf("Error: Failed to load config: %v\n", err)
			os.Exit(1)
		}

		if err := store.SaveConfig(target, configData); err != nil {
			cmd.Printf("Error: Failed to save config: %v\n", err)
			os.Exit(1)
		}

		// Copy layers directory
		sourceLayersDir := filepath.Join(storageDir, "images",
			strings.ReplaceAll(strings.ReplaceAll(source, ":", "_"), "/", "_"), "layers")
		targetLayersDir := filepath.Join(storageDir, "images",
			strings.ReplaceAll(strings.ReplaceAll(target, ":", "_"), "/", "_"), "layers")

		if err := os.MkdirAll(filepath.Dir(targetLayersDir), 0755); err != nil {
			cmd.Printf("Error: Failed to create target directory: %v\n", err)
			os.Exit(1)
		}

		// Copy all layer files
		files, err := os.ReadDir(sourceLayersDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() {
					srcFile := filepath.Join(sourceLayersDir, file.Name())
					dstFile := filepath.Join(targetLayersDir, file.Name())

					if err := copyFile(srcFile, dstFile); err != nil {
						cmd.Printf("Error: Failed to copy layer: %v\n", err)
						os.Exit(1)
					}
				}
			}
		}

		cmd.Printf("Tagged %s as %s\n", source, target)
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return nil
}
