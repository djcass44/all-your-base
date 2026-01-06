package alpine

import (
	"context"
	"path/filepath"
	"strings"

	"chainguard.dev/apko/pkg/apk/apk"
	"chainguard.dev/apko/pkg/apk/fs"
	"github.com/djcass44/all-your-base/pkg/packages"
)

var installedFiles = []string{
	filepath.Join("/lib", "apk", "db", "installed"),
	filepath.Join("/usr", "lib", "apk", "db", "installed"),
}

// writeInstalled updates an Alpine "installed packages" database to include
// a given package.
func (p *PackageKeeper) writeInstalled(ctx context.Context, pkg []*apk.RepositoryPackage, rootfs fs.FullFS) error {
	return packages.RecordAll(ctx, p.base, installedFiles, pkg, rootfs, func(t *apk.RepositoryPackage) string {
		return strings.Join(apk.PackageToInstalled(t.Package), "\n")
	})
}
