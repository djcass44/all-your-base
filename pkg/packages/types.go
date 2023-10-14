package packages

import (
	"context"

	"github.com/chainguard-dev/go-apk/pkg/fs"
	"github.com/djcass44/all-your-base/pkg/lockfile"
)

type PackageManager interface {
	Unpack(ctx context.Context, pkg string, rootfs fs.FullFS) error
	Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error)
	Record(ctx context.Context, pkg string, rootfs fs.FullFS) error
}
