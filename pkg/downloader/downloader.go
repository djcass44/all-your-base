package downloader

import (
	"context"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-getter/v2"
)

func NewDownloader(cacheDir string) (*Downloader, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}
	return &Downloader{cacheDir: cacheDir}, nil
}

// Download retrieves a file from the given 'src' and stores
// it in the cache directory.
func (d *Downloader) Download(ctx context.Context, src string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)

	uri, err := url.Parse(src)
	if err != nil {
		log.Error(err, "failed to parse url", "src", src)
		return "", err
	}

	// download the file to a predictable location so that
	// we can avoid repeated downloads
	dst := filepath.Join(d.cacheDir, filepath.Base(uri.Path))
	log.V(1).Info("downloading file", "src", src, "dst", dst)

	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		log.V(1).Info("skipping file download as it already exists", "src", src, "dst", dst)
		return dst, nil
	}

	req := &getter.Request{
		Src:             src,
		Dst:             dst,
		GetMode:         getter.ModeFile,
		DisableSymlinks: true,
	}
	if _, err := getter.DefaultClient.Get(ctx, req); err != nil {
		log.Error(err, "failed to download file", "src", src)
		return "", err
	}
	// we need to chmod the files so that the root group
	// can access them as if they were the owner
	if err := os.Chmod(dst, 0664); err != nil {
		log.Error(err, "failed to update file permissions", "file", dst)
		return "", err
	}

	return dst, nil
}
