package debian

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/archiveutil"
	"github.com/djcass44/all-your-base/pkg/debian"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/go-logr/logr"
	"os"
	"strings"
)

type PackageKeeper struct {
	indices []*debian.Index
}

func NewPackageKeeper(ctx context.Context, repositories []string) (*PackageKeeper, error) {
	log := logr.FromContextOrDiscard(ctx)

	var indices []*debian.Index
	for _, repo := range repositories {
		bits := strings.Split(repo, " ")
		if len(bits) != 3 {
			return nil, fmt.Errorf("malformed repository url, expecting: 'base release component'")
		}
		idx, err := debian.NewIndex(ctx, bits[0], bits[1], bits[2], "amd64")
		if err != nil {
			return nil, err
		}
		log.V(2).Info("added index", "count", idx.Count(), "source", repo)
		indices = append(indices, idx)
	}
	return &PackageKeeper{
		indices: indices,
	}, nil
}

func (p *PackageKeeper) Unpack(ctx context.Context, pkg string, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkg)
	log.Info("unpacking deb")

	// first we need to unpack the deb file using
	// the equivalent of 'ar -x'
	f, err := os.Open(pkg)
	if err != nil {
		log.Error(err, "failed to open file")
		return err
	}
	defer f.Close()

	tmpFs := fs.NewMemFS()

	if err := archiveutil.Unar(ctx, f, tmpFs); err != nil {
		return err
	}

	// then we need to unpack the 'data.tar.X' file
	// that contains the filesystem

	if _, err := tmpFs.Stat(dataXZ); err == nil {
		return p.unpackXZ(ctx, tmpFs, rootfs)
	}
	if _, err := tmpFs.Stat(dataZstd); err == nil {
		return p.unpackZstd(ctx, tmpFs, rootfs)
	}

	return errors.New("unknown or unsupported data archive")
}

const (
	dataXZ   = "/data.tar.xz"
	dataZstd = "/data.tar.zst"
)

func (*PackageKeeper) unpackZstd(ctx context.Context, src fs.FullFS, dst fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx)
	log.V(1).Info("unpacking zstandard data archive")
	f, err := src.Open(dataZstd)
	if err != nil {
		log.Error(err, "failed to open data.tar.xz file")
		return err
	}
	return archiveutil.Zuntar(ctx, f, dst)
}

func (*PackageKeeper) unpackXZ(ctx context.Context, src fs.FullFS, dst fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx)
	log.V(1).Info("unpacking xz data archive")
	f, err := src.Open(dataXZ)
	if err != nil {
		log.Error(err, "failed to open data.tar.xz file")
		return err
	}
	return archiveutil.XZuntar(ctx, f, dst)
}

func (p *PackageKeeper) Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error) {
	for _, idx := range p.indices {
		out, err := idx.GetPackageWithDependencies(ctx, map[string]debian.Package{}, &debian.PackageVersion{
			Names: []string{pkg},
		})
		if err != nil {
			return nil, err
		}
		if len(out) == 0 {
			continue
		}
		names := make([]lockfile.Package, len(out))
		for i := range out {
			names[i] = lockfile.Package{
				Name:      out[i].Package,
				Resolved:  strings.TrimSuffix(idx.Source(), "/") + "/" + strings.TrimPrefix(out[i].Filename, "/"),
				Integrity: out[i].Sha256,
				Version:   out[i].Version,
				Type:      v1.PackageDebian,
			}
		}
		return names, nil
	}
	return nil, fmt.Errorf("package could not be found in any index: %s", pkg)
}
