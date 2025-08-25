package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Removes all cached file downloads",
	RunE:  clean,
}

const (
	flagCacheDir = "cache-dir"
)

func init() {
	cleanCmd.Flags().String(flagCacheDir, "", "cache directory (defaults to user cache dir)")
}

func clean(cmd *cobra.Command, _ []string) error {
	log := logr.FromContextOrDiscard(cmd.Context())

	cacheDir, _ := cmd.Flags().GetString(flagCacheDir)
	cacheDir = getCacheDir(cacheDir)

	log.Info("deleting cache dir", "dir", cacheDir)

	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("removing cache dir: %w", err)
	}
	return nil
}

func getCacheDir(d string) string {
	if d == "" {
		d, _ = os.UserCacheDir()
		d = filepath.Join(d, "ayb")
	}
	return filepath.Clean(d)
}
