package alpine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/chainguard-dev/go-apk/pkg/apk"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/archiveutil"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/go-logr/logr"
)

var worldFile = filepath.Join("etc", "apk", "world")

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

func (p *PackageKeeper) Record(ctx context.Context, pkg string, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg)

	world, err := rootfs.ReadFile(worldFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error(err, "failed to open world file")
		return err
	}

	// check if the package is already in the world file
	scanner := bufio.NewScanner(bytes.NewReader(world))
	for scanner.Scan() {
		line := scanner.Text()
		if line == pkg {
			log.V(2).Info("located package in world file")
			return nil
		}

	}
	if err := scanner.Err(); err != nil {
		log.Error(err, "failed to read world file")
		return err
	}

	// otherwise, append and write
	if err := rootfs.MkdirAll(filepath.Dir(worldFile), 0755); err != nil {
		log.Error(err, "failed to create world directory")
		return err
	}

	log.V(1).Info("appending to the world file")
	f, err := rootfs.OpenFile(worldFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err, "failed to open world file for writing")
		return err
	}
	defer f.Close()

	if _, err = f.Write([]byte(pkg + "\n")); err != nil {
		log.Error(err, "failed to write to world file")
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
		Direct:    true,
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
