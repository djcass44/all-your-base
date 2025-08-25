package statements

import (
	"fmt"

	cbev1 "github.com/Snakdy/container-build-engine/pkg/api/v1"
	"github.com/Snakdy/container-build-engine/pkg/pipelines"
	"github.com/Snakdy/container-build-engine/pkg/pipelines/utils"
	"github.com/djcass44/all-your-base/pkg/airutil"
	aybv1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/downloader"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/packages"
	"github.com/djcass44/all-your-base/pkg/packages/alpine"
	"github.com/djcass44/all-your-base/pkg/packages/debian"
	"github.com/djcass44/all-your-base/pkg/packages/rpm"
	"github.com/go-logr/logr"
)

func NewPackageStatement(alpineKeeper *alpine.PackageKeeper, debianKeeper *debian.PackageKeeper, yumKeeper *rpm.PackageKeeper, dl *downloader.Downloader, forceChecksum bool) *PackageStatement {
	return &PackageStatement{
		alpineKeeper:  alpineKeeper,
		debianKeeper:  debianKeeper,
		yumKeeper:     yumKeeper,
		dl:            dl,
		forceChecksum: forceChecksum,
	}
}

func (s *PackageStatement) Run(ctx *pipelines.BuildContext, _ ...cbev1.Options) (cbev1.Options, error) {
	log := logr.FromContextOrDiscard(ctx.Context)

	packageType, err := cbev1.GetRequired[string](s.options, "type")
	if err != nil {
		return cbev1.Options{}, err
	}
	name, err := cbev1.GetRequired[string](s.options, "name")
	if err != nil {
		return cbev1.Options{}, err
	}
	version, err := cbev1.GetRequired[string](s.options, "version")
	if err != nil {
		return cbev1.Options{}, err
	}
	resolved, err := cbev1.GetRequired[string](s.options, "resolved")
	if err != nil {
		return cbev1.Options{}, err
	}
	checksum, err := cbev1.GetOptional[string](s.options, "checksum")
	if err != nil {
		return cbev1.Options{}, err
	}

	var keeper packages.PackageManager
	switch aybv1.PackageType(packageType) {
	case aybv1.PackageAlpine:
		keeper = s.alpineKeeper
	case aybv1.PackageDebian:
		keeper = s.debianKeeper
	case aybv1.PackageRPM:
		keeper = s.yumKeeper
	case aybv1.PackageOCI:
		fallthrough
	case aybv1.PackageDir:
		fallthrough
	case aybv1.PackageFile:
		return cbev1.Options{}, nil
	default:
		return cbev1.Options{}, fmt.Errorf("unknown package type: %s", packageType)
	}

	log.V(1).Info("installing package", "name", name, "version", version)

	// download the package
	pkgPath, err := s.dl.Download(ctx.Context, airutil.ExpandEnv(resolved))
	if err != nil {
		return cbev1.Options{}, err
	}

	// generate and validate the checksum
	// of the package
	actualChecksum, err := lockfile.HashFile(pkgPath)
	if err != nil {
		return cbev1.Options{}, err
	}
	actualChecksum = "sha256:" + actualChecksum
	if checksum != "" || s.forceChecksum {
		log.V(1).Info("verifying package checksum", "name", name, "version", version, "expectedChecksum", checksum, "actualChecksum", actualChecksum)
		if checksum != actualChecksum {
			log.Info("package checksum mismatch detected - consider clearing the cache using 'ayb cache clean'")
			return cbev1.Options{}, fmt.Errorf("package checksum mismatch: expected '%s', actual '%s'", checksum, actualChecksum)
		}
	}

	// unpack the package into the root
	// filesystem
	if err := keeper.Unpack(ctx.Context, pkgPath, ctx.FS); err != nil {
		return cbev1.Options{}, err
	}
	return cbev1.Options{}, nil
}

func (*PackageStatement) Name() string {
	return StatementPackage
}

func (*PackageStatement) MutatesConfig() bool {
	return true
}

func (*PackageStatement) MutatesFS() bool {
	return false
}

func (s *PackageStatement) SetOptions(options cbev1.Options) {
	if s.options == nil {
		s.options = map[string]any{}
	}
	utils.CopyMap(options, s.options)
}
