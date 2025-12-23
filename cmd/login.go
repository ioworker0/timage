package cmd

import (
	"os"

	"github.com/ioworker0/timage/pkg/config"
	"github.com/ioworker0/timage/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	loginUsername string
	loginPassword string
)

var loginCmd = &cobra.Command{
	Use:   "login [registry]",
	Short: "Login to a registry",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		registryURL := "docker.io"
		if len(args) > 0 {
			registryURL = args[0]
		}

		// Get config directory
		configDir, err := config.GetConfigDir()
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Load config
		cfg, err := config.NewManager(configDir)
		if err != nil {
			cmd.Printf("Error: Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Create auth config
		auth := &registry.AuthConfig{
			Username: loginUsername,
			Password: loginPassword,
		}

		// Create registry client to verify credentials
		client, err := registry.NewClient(registryURL, auth, "")
		if err != nil {
			cmd.Printf("Error: Failed to create registry client: %v\n", err)
			os.Exit(1)
		}

		// Verify credentials
		cmd.Printf("Verifying credentials for %s...\n", registryURL)
		if err := client.VerifyCredentials(); err != nil {
			cmd.Printf("Error: Authentication failed: %v\n", err)
			cmd.Printf("\nNote: Make sure your username and password are correct.\n")
			cmd.Printf("For Harbor, you might need to use your email as username.\n")
			os.Exit(1)
		}

		// Save credentials
		cfg.SetAuth(registryURL, loginUsername, loginPassword)

		if err := cfg.Save(); err != nil {
			cmd.Printf("Error: Failed to save config: %v\n", err)
			os.Exit(1)
		}

		cmd.Printf("Login succeeded for %s\n", registryURL)
	},
}

func init() {
	loginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Username")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Password")
	loginCmd.MarkFlagRequired("username")
	loginCmd.MarkFlagRequired("password")
	rootCmd.AddCommand(loginCmd)
}
