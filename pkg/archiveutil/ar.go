package archiveutil

import (
	"context"
	"github.com/blakesmith/ar"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/go-logr/logr"
	"io"
	"os"
	"path/filepath"
)

// Unar expands an ar archive into the given path.
func Unar(ctx context.Context, r io.Reader, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx)
	tr := ar.NewReader(r)

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

		target := filepath.Clean("/" + header.Name)

		log.V(5).Info("creating file", "target", target, "mode", header.Mode)
		f, err := rootfs.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
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
