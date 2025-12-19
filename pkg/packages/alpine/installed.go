package alpine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"chainguard.dev/apko/pkg/apk/apk"
	"chainguard.dev/apko/pkg/apk/fs"
	"github.com/go-logr/logr"
)

// writeInstalled updates an Alpine "installed packages" database to include
// a given package.
func (p *PackageKeeper) writeInstalled(ctx context.Context, path string, pkg *apk.RepositoryPackage, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg.Name, "path", path)
	log.V(5).Info("recording package")

	path = filepath.Clean(path)

	world, err := rootfs.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error(err, "failed to open installed file")
		return err
	}

	// check if the package is already in the world file
	scanner := bufio.NewScanner(bytes.NewReader(world))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "P:"+pkg.Name {
			log.V(5).Info("located package in installed file")
			return nil
		}

	}
	if err := scanner.Err(); err != nil {
		log.Error(err, "failed to read world file")
		return err
	}

	// otherwise, append and write
	if err := rootfs.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Error(err, "failed to create world directory")
		return err
	}

	log.V(5).Info("appending to the installed file")
	f, err := rootfs.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err, "failed to open installed file for writing")
		return err
	}
	defer f.Close()

	out := apk.PackageToInstalled(pkg.Package)

	if _, err = f.Write([]byte(strings.Join(out, "\n") + "\n")); err != nil {
		log.Error(err, "failed to write to installed file")
		return err
	}

	return nil
}
