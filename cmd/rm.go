package cmd

import (
	"os"

	"github.com/ioworker0/timage/pkg/config"
	"github.com/ioworker0/timage/pkg/storage"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm [image]",
	Short: "Remove a local image",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		imageRef := args[0]

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

		// Check if image exists
		if !store.ImageExists(imageRef) {
			cmd.Printf("Error: Image '%s' not found\n", imageRef)
			os.Exit(1)
		}

		// Remove image
		if err := store.RemoveImage(imageRef); err != nil {
			cmd.Printf("Error: Failed to remove image: %v\n", err)
			os.Exit(1)
		}

		cmd.Printf("Removed: %s\n", imageRef)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
