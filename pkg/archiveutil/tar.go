package archiveutil

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"github.com/go-logr/logr"
	"io"
	"os"
	"path/filepath"
)

// Guntar is the same as Untar, but it first decodes the gzipped archive.
func Guntar(ctx context.Context, r io.Reader, path string) error {
	gzp, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzp.Close()
	return Untar(ctx, gzp, path)
}

// Untar expands a tar archive into the given path.
func Untar(ctx context.Context, r io.Reader, path string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("path", path)
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			log.Error(err, "failed to read file from archive")
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(path, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			log.V(5).Info("creating directory", "target", target)
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					log.Error(err, "failed to create directory", "target", target)
					return err
				}
			}
		case tar.TypeReg:
			log.V(5).Info("creating file", "target", target, "mode", header.Mode)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				log.Error(err, "failed to open file", "target", target)
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				log.Error(err, "failed to extract file", "target", target)
				_ = f.Close()
				return err
			}
			_ = f.Close()
		}
	}
}
