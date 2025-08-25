package lockfile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/carlmjohnson/requests"
	"github.com/gosimple/hashdir"
)

func HashFile(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashDir(path string) (string, error) {
	return hashdir.Make(path, "sha256")
}

func HashURL(ctx context.Context, path string) (string, error) {
	tmpf, err := os.CreateTemp("", "lockfile-")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	_ = tmpf.Close()
	defer func() {
		_ = os.Remove(tmpf.Name())
	}()
	// download the file
	if err := requests.URL(path).ToFile(tmpf.Name()).Fetch(ctx); err != nil {
		return "", fmt.Errorf("downloading file: %w", err)
	}
	// return the hash of the file
	return HashFile(tmpf.Name())
}
