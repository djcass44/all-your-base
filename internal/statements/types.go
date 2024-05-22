package statements

import (
	cbev1 "github.com/Snakdy/container-build-engine/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/downloader"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/djcass44/all-your-base/pkg/packages/debian"
	"github.com/djcass44/all-your-base/pkg/packages/rpm"
)

const (
	StatementPackage = "package"
	StatementEnv     = "set-env"
)

type PackageStatement struct {
	options      cbev1.Options
	alpineKeeper *alpine.PackageKeeper
	debianKeeper *debian.PackageKeeper
	yumKeeper    *rpm.PackageKeeper
	dl           *downloader.Downloader
}
