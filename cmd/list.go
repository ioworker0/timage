package cmd

import (
	"os"

	"github.com/ioworker0/timage/pkg/config"
	"github.com/ioworker0/timage/pkg/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local images",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		// Get storage directory
		storageDir, err := config.GetStorageDir()
		if err != nil {
			cmd.Printf("Error: Failed to get storage directory: %v\n", err)
			os.Exit(1)
		}

		// Create store
		store, err := storage.NewStore(storageDir)
		if err != nil {
			cmd.Printf("Error: Failed to create store: %v\n", err)
			os.Exit(1)
		}

		// List images
		images, err := store.ListImages()
		if err != nil {
			cmd.Printf("Error: Failed to list images: %v\n", err)
			os.Exit(1)
		}

		// Display images
		if len(images) == 0 {
			cmd.Println("No images found")
			return
		}

		cmd.Println("Local images:")
		for _, image := range images {
			cmd.Printf("  %s\n", image)
		}

		cmd.Printf("\nTotal: %d image(s)\n", len(images))
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
