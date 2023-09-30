package archiveutil

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/go-logr/logr"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Guntar is the same as Untar, but it first decodes the gzipped archive.
func Guntar(ctx context.Context, r io.Reader, rootfs fs.FullFS) error {
	gzp, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzp.Close()
	return Untar(ctx, gzp, rootfs)
}

// XZuntar is the same as Untar, but it first decodes the XZ archive
func XZuntar(ctx context.Context, r io.Reader, rootfs fs.FullFS) error {
	xzp, err := xz.NewReader(r)
	if err != nil {
		return err
	}
	return Untar(ctx, xzp, rootfs)
}

// Zuntar is the same as Untar, but it first decodes the Zstandard archive
func Zuntar(ctx context.Context, r io.Reader, rootfs fs.FullFS) error {
	zp, err := zstd.NewReader(r)
	if err != nil {
		return err
	}
	defer zp.Close()
	return Untar(ctx, zp, rootfs)
}

// Untar expands a tar archive into the given path.
func Untar(ctx context.Context, r io.Reader, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx)
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

		target := filepath.Clean("/" + header.Name)

		switch header.Typeflag {
		default:
			log.V(4).Info("unexpected header type", "target", target, "type", header.Typeflag)
		case tar.TypeDir:
			log.V(5).Info("creating directory", "target", target)
			if _, err := rootfs.Stat(target); err != nil {
				if err := rootfs.MkdirAll(target, 0755); err != nil {
					log.Error(err, "failed to create directory", "target", target)
					return err
				}
			}
		case tar.TypeLink:
			safeSrc := strings.TrimPrefix(header.Linkname, ".")
			log.V(5).Info("creating hard link", "target", target, "source", safeSrc)
			if err := rootfs.Link(safeSrc, target); err != nil {
				if errors.Is(err, os.ErrExist) {
					log.V(5).Info("skipping hard link as target file already exists")
					continue
				}
				return err
			}
		case tar.TypeSymlink:
			oldname := filepath.Join(filepath.Dir(target), header.Linkname)
			if filepath.IsAbs(header.Linkname) {
				oldname = header.Linkname
			}
			log.V(5).Info("creating symbolic link", "target", target, "source", oldname)
			if err := rootfs.Symlink(oldname, target); err != nil {
				if errors.Is(err, os.ErrExist) {
					log.V(5).Info("skipping symbolic link as target file already exists")
					continue
				}
				return err
			}
		case tar.TypeReg:
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
}
