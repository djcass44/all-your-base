package rpm

import (
	"context"
	"fmt"
	"github.com/cavaliergopher/rpm"
	"github.com/chainguard-dev/go-apk/pkg/fs"
	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"github.com/djcass44/all-your-base/pkg/lockfile"
	"github.com/djcass44/all-your-base/pkg/yum"
	"github.com/djcass44/all-your-base/pkg/yum/yumindex"
	"github.com/go-logr/logr"
	"github.com/sassoftware/go-rpmutils/cpio"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type PackageKeeper struct {
	indices []*yumindex.Metadata
}

const (
	ModeRegular            = 33188
	ModeBinary             = 33261
	ModeSomethingMaybeJson = 41471
)

var allowedModes = []int{ModeRegular, ModeBinary, ModeSomethingMaybeJson}

func NewPackageKeeper(ctx context.Context, repositories []string) (*PackageKeeper, error) {
	log := logr.FromContextOrDiscard(ctx)

	var indices []*yumindex.Metadata
	for _, repo := range repositories {
		idx, err := yum.NewIndex(ctx, repo)
		if err != nil {
			return nil, err
		}
		log.V(2).Info("added index", "count", idx.Packages, "source", repo)
		indices = append(indices, idx)
	}
	return &PackageKeeper{
		indices: indices,
	}, nil
}

func (p *PackageKeeper) Unpack(ctx context.Context, pkgFile string, rootfs fs.FullFS) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("pkg", pkgFile)
	log.Info("unpacking rpm")

	f, err := os.Open(pkgFile)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	pkg, err := rpm.Read(f)
	if err != nil {
		return fmt.Errorf("reading package header: %w", err)
	}

	if compression := pkg.PayloadCompression(); compression != "xz" {
		return fmt.Errorf("unsupported compression: %s", compression)
	}

	xzReader, err := xz.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating xz reader: %w", err)
	}

	if format := pkg.PayloadFormat(); format != "cpio" {
		return fmt.Errorf("unsupported payload format: %s", format)
	}

	cpioReader := cpio.NewReader(xzReader)
	for {
		hdr, err := cpioReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading cpio: %w", err)
		}
		// skip non-files
		if !slices.Contains(allowedModes, hdr.Mode()) {
			log.V(4).Info("skipping non-file header", "path", hdr.Filename(), "mode", hdr.Mode())
			continue
		}
		// create the target directory
		if dir := filepath.Dir(hdr.Filename()); dir != "" {
			log.V(6).Info("creating directory", "dir", dir)
			if err := rootfs.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}
		}
		log.V(5).Info("creating file", "file", hdr.Filename())
		out, err := rootfs.Create(hdr.Filename())
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}
		if _, err := io.Copy(out, cpioReader); err != nil {
			_ = out.Close()
			return fmt.Errorf("copying file: %w", err)
		}
		_ = out.Close()
	}

	return nil
}

func (p *PackageKeeper) Resolve(ctx context.Context, pkg string) ([]lockfile.Package, error) {
	for _, idx := range p.indices {
		for _, p := range idx.PackagesList {
			if p.Name == pkg {
				return []lockfile.Package{
					{
						Name:      p.Name,
						Type:      v1.PackageRPM,
						Version:   p.Version.Ver,
						Resolved:  strings.TrimSuffix(idx.Source, "/") + "/" + strings.TrimPrefix(p.Location.Href, "/"),
						Integrity: p.Checksum.Value,
						Direct:    true,
					},
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("package could not be found in any index: %s", pkg)
}
