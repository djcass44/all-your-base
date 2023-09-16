package packages

import "context"

type Package interface {
	Unpack(ctx context.Context, pkg, rootfs string) error
	Resolve(ctx context.Context, pkg string) ([]string, error)
}
