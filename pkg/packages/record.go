package packages

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"chainguard.dev/apko/pkg/apk/fs"
	"github.com/djcass44/all-your-base/pkg/archiveutil"
	"github.com/go-logr/logr"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

func RecordAll[T any](ctx context.Context, base v1.Image, paths []string, packages []T, rootfs fs.FullFS, format func(t T) string) error {
	for _, path := range paths {
		if err := Record(ctx, base, path, packages, rootfs, format); err != nil {
			return fmt.Errorf("recording %s: %w", path, err)
		}
	}
	return nil
}

func Record[T any](ctx context.Context, base v1.Image, path string, packages []T, rootfs fs.FullFS, format func(t T) string) error {
	if base == nil {
		return nil
	}

	log := logr.FromContextOrDiscard(ctx).WithValues("path", path)
	log.V(5).Info("recording packages")

	path = filepath.Clean(path)

	if err := archiveutil.UntarFile(ctx, mutate.Extract(base), path, rootfs); err != nil {
		return fmt.Errorf("extracting file from base image '%s': %w", path, err)
	}

	// otherwise, append and write
	if err := rootfs.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	log.V(5).Info("appending to the package record")
	f, err := rootfs.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file '%s': %w", path, err)
	}
	defer f.Close()

	out := strings.Builder{}
	for _, pkg := range packages {
		log.V(5).Info("recording package", "pkg", pkg)
		out.WriteString(format(pkg))
		out.WriteString("\n")
	}

	if _, err = f.Write([]byte(out.String() + "\n")); err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	return nil
}
