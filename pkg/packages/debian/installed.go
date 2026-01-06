package debian

import (
	"context"
	"fmt"
	"net/textproto"
	"path/filepath"
	"strings"

	"chainguard.dev/apko/pkg/apk/fs"
	"github.com/djcass44/all-your-base/pkg/debian"
	"github.com/djcass44/all-your-base/pkg/packages"
)

var installedFiles = []string{
	filepath.Join("/var", "lib", "dpkg", "status"),
}

// writeInstalled updates an Alpine "installed packages" database to include
// a given package.
func (p *PackageKeeper) writeInstalled(ctx context.Context, pkg []debian.Package, rootfs fs.FullFS) error {
	return packages.RecordAll(ctx, p.base, installedFiles, pkg, rootfs, func(t debian.Package) string {
		return packageToInstalled(&t)
	})
}

func packageToInstalled(pkg *debian.Package) string {
	block := textproto.MIMEHeader{}
	block.Set("Package", pkg.Package)
	block.Set("Version", pkg.Version)
	block.Set("Architecture", pkg.Architecture)
	block.Set("Depends", strings.Join(pkg.Depends, ", "))
	block.Set("Status", "install ok installed")

	sb := strings.Builder{}
	for k := range block {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, block.Get(k)))
	}
	return sb.String()
}
