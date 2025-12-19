package alpine

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"chainguard.dev/apko/pkg/apk/apk"
	"chainguard.dev/apko/pkg/apk/fs"

	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/archiveutil"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/go-logr/logr"
)

var installedFiles = []string{
	filepath.Join("/lib", "apk", "db", "installed"),
	filepath.Join("/usr", "lib", "apk", "db", "installed"),
}

type PackageKeeper struct {
	rootfs  fs.FullFS
	indices []apk.NamedIndex
}

func NewPackageKeeper(ctx context.Context, repositories []string, rootfs fs.FullFS) (*PackageKeeper, error) {
	log := logr.FromContextOrDiscard(ctx)
	indices, err := apk.GetRepositoryIndexes(ctx, repositories, map[string][]byte{}, "x86_64", apk.WithIgnoreSignatures(true), apk.WithHTTPClient(http.DefaultClient))
	if err != nil {
		return nil, err
	}

	log.V(2).Info("loaded indices", "count", len(indices))
	for _, i := range indices {
		log.V(1).Info("added index", "count", i.Count(), "name", i.Name(), "source", i.Source())
	}

	return &PackageKeeper{
		indices: indices,
		rootfs:  rootfs,
	}, nil
}

func (*PackageKeeper) Unpack(ctx context.Context, pkg string, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg)
	log.V(4).Info("unpacking apk")

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

func (p *PackageKeeper) Record(ctx context.Context, pkg *apk.RepositoryPackage, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg.Name)
	log.V(5).Info("recording package")

	for _, i := range installedFiles {
		if err := p.writeInstalled(ctx, i, pkg, rootfs); err != nil {
			return err
		}
	}

	return nil
}

func (p *PackageKeeper) Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error) {
	resolver := apk.NewPkgResolver(ctx, p.indices)

	// resolve the package
	dq := map[*apk.RepositoryPackage]string{}
	repoPkg, repoPkgDeps, _, err := resolver.GetPackageWithDependencies(ctx, pkg, nil, dq)
	if err != nil {
		return nil, err
	}

	if err := p.Record(ctx, repoPkg, p.rootfs); err != nil {
		return nil, err
	}

	// collect the urls for each package
	names := make([]lockfile.Package, len(repoPkgDeps)+1)
	names[0] = lockfile.Package{
		Name:      repoPkg.Name,
		Resolved:  repoPkg.URL(),
		Integrity: repoPkg.ChecksumString(),
		Version:   repoPkg.Version,
		Type:      v1.PackageAlpine,
		Direct:    true,
	}
	for i := range repoPkgDeps {
		if err := p.Record(ctx, repoPkgDeps[i], p.rootfs); err != nil {
			return nil, err
		}
		names[i+1] = lockfile.Package{
			Name:      repoPkgDeps[i].Name,
			Resolved:  repoPkgDeps[i].URL(),
			Integrity: repoPkgDeps[i].ChecksumString(),
			Version:   repoPkgDeps[i].Version,
			Type:      v1.PackageAlpine,
		}
	}

	return names, nil
}
