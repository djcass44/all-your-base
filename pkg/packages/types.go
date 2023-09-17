package packages

import (
	"context"
	"github.com/djcass44/all-your-base/pkg/lockfile"
)

type PackageManager interface {
	Unpack(ctx context.Context, pkg, rootfs string) error
	Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error)
}
