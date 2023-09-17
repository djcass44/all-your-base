package downloader

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-getter"
	"net/url"
	"os"
	"path/filepath"
)

func NewDownloader(cacheDir string) (*Downloader, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}
	return &Downloader{cacheDir: cacheDir}, nil
}

func (d *Downloader) Download(ctx context.Context, src string) (string, error) {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("downloading file", "src", src)

	uri, err := url.Parse(src)
	if err != nil {
		log.Error(err, "failed to parse url")
		return "", err
	}

	// download the file to a predictable location so that
	// we can avoid repeated downloads
	dst := filepath.Join(d.cacheDir, filepath.Base(uri.Path))
	log.V(1).Info("preparing to download file", "dst", dst)

	client := &getter.Client{
		Ctx:             ctx,
		Src:             src,
		Dst:             dst,
		Mode:            getter.ClientModeFile,
		DisableSymlinks: true,
	}
	if err := client.Get(); err != nil {
		log.Error(err, "failed to download file")
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
