package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "timage",
	Short: "A Docker image management CLI tool",
	Long: `Timage is a completely independent Docker image management CLI tool.
It communicates directly with Docker Registry API v2 to pull, push, tag,
and remove images without any local Docker dependency.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// 全局 flag
	rootCmd.PersistentFlags().StringP("proxy", "x", "", "Proxy URL (e.g., http://127.0.0.1:7890, socks5://127.0.0.1:1080)")
}
