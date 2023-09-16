package alpine

import (
	"context"
	"github.com/djcass44/all-your-base/pkg/archiveutil"
	"github.com/go-logr/logr"
	"os"
)

type PackageKeeper struct{}

func (p *PackageKeeper) Unpack(ctx context.Context, pkg, rootfs string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg, "rootfs", rootfs)
	log.Info("unpacking apk")

	f, err := os.Open(pkg)
	if err != nil {
		log.Error(err, "failed to open file")
		return err
	}
	defer f.Close()

	if err := archiveutil.Guntar(ctx, f, rootfs); err != nil {
		return err
	}
	return nil
}
