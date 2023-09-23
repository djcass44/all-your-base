package alpine

import (
	"context"
	"github.com/chainguard-dev/go-apk/pkg/apk"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/archiveutil"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/go-logr/logr"
	"os"
)

type PackageKeeper struct {
	indices []apk.NamedIndex
}

func NewPackageKeeper(ctx context.Context, repositories []string) (*PackageKeeper, error) {
	log := logr.FromContextOrDiscard(ctx)
	indices, err := apk.GetRepositoryIndexes(ctx, repositories, map[string][]byte{}, "x86_64", apk.WithIgnoreSignatures(true))
	if err != nil {
		return nil, err
	}

	log.V(2).Info("loaded indices", "count", len(indices))
	for _, i := range indices {
		log.V(1).Info("added index", "count", i.Count(), "name", i.Name(), "source", i.Source())
	}

	return &PackageKeeper{
		indices: indices,
	}, nil
}

func (*PackageKeeper) Unpack(ctx context.Context, pkg string, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg)
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

func (p *PackageKeeper) Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error) {
	resolver := apk.NewPkgResolver(ctx, p.indices)

	// resolve the package
	repoPkg, repoPkgDeps, _, err := resolver.GetPackageWithDependencies(pkg, nil)
	if err != nil {
		return nil, err
	}

	// collect the urls for each package
	names := make([]lockfile.Package, len(repoPkgDeps)+1)
	names[0] = lockfile.Package{
		Name:      repoPkg.Name,
		Resolved:  repoPkg.Url(),
		Integrity: repoPkg.ChecksumString(),
		Version:   repoPkg.Version,
		Type:      v1.PackageAlpine,
	}
	for i := range repoPkgDeps {
		names[i+1] = lockfile.Package{
			Name:      repoPkgDeps[i].Name,
			Resolved:  repoPkgDeps[i].Url(),
			Integrity: repoPkgDeps[i].ChecksumString(),
			Version:   repoPkgDeps[i].Version,
			Type:      v1.PackageAlpine,
		}
	}

	return names, nil
}
