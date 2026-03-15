package packages

import (
	"context"

	"chainguard.dev/apko/pkg/apk/fs"

	"github.com/djcass44/all-your-base/pkg/lockfile"
)

type PackageManager interface {
	Unpack(ctx context.Context, pkg string, rootfs fs.FullFS) error
	Resolve(ctx context.Context, pkg string, write bool) ([]lockfile.Package, error)
}
