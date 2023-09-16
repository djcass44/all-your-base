package packages

import "context"

type Package interface {
	Unpack(ctx context.Context, pkg, rootfs string) error
}
