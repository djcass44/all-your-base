package debian

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	version "github.com/knqyf263/go-deb-version"
	"io"
	"net/http"
	"os"
	"pault.ag/go/debian/control"
	"slices"
)

func NewIndex(ctx context.Context, repository, release, component, arch string) (*Index, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("repo", repository, "release", release, "component", component, "arch", arch)
	log.V(1).Info("downloading index")

	target := fmt.Sprintf("%s/dists/%s/%s/binary-%s/Packages.gz", repository, release, component, arch)
	f, err := os.CreateTemp("", "Packages-*.gz")
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(target)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Info("failed to locate Packages.gz file", "url", target)
		return nil, fmt.Errorf("http response failed with code: %d", resp.StatusCode)
	}
	log.V(1).Info("successfully downloaded index", "code", resp.StatusCode)
	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	if _, err := io.Copy(f, gr); err != nil {
		return nil, err
	}
	_ = f.Close()

	return newIndex(ctx, repository, f.Name())
}

func newIndex(ctx context.Context, source, path string) (*Index, error) {
	log := logr.FromContextOrDiscard(ctx).WithValues("source", source, "path", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	dec, err := control.NewDecoder(f, nil)
	if err != nil {
		return nil, err
	}
	var out []Package
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	log.V(1).Info("successfully decoded index", "count", len(out))
	return &Index{
		packages: out,
		source:   source,
	}, nil
}

func (idx *Index) Count() int {
	return len(idx.packages)
}

func (idx *Index) Source() string {
	return idx.source
}

func (idx *Index) GetPackageWithDependencies(ctx context.Context, existing map[string]Package, pv *PackageVersion) ([]Package, error) {
	log := logr.FromContextOrDiscard(ctx)
	for _, p := range idx.packages {
		if slices.Contains(pv.Names, p.Package) && pv.Matches(p.Version) {
			log.V(5).Info("found package match", "name", p.Package, "version", p.Version, "deps", len(p.Depends))
			// skip duplicate packages
			if _, ok := existing[p.String()]; ok {
				log.V(4).Info("skipping package as we already have it", "name", p.Package, "version", p.Version)
				continue
			}
			existing[p.String()] = p
			for _, dep := range p.Depends {
				dv, err := ParseVersion(dep)
				if err != nil {
					return nil, err
				}
				deps, err := idx.GetPackageWithDependencies(ctx, existing, dv)
				if err != nil {
					return nil, err
				}
				for _, d := range deps {
					existing[d.String()] = d
				}
			}
			var pkg []Package
			for _, v := range existing {
				pkg = append(pkg, v)
			}
			return pkg, nil
		}
	}
	return nil, nil
}

func (pv *PackageVersion) Matches(s1 string) bool {
	// if there's a version missing, match
	// anything
	if s1 == "" || pv.Version == "" {
		return true
	}
	v1, err := version.NewVersion(s1)
	if err != nil {
		return false
	}
	v2, err := version.NewVersion(pv.Version)
	if err != nil {
		return false
	}
	switch pv.Constraint {
	case ">>":
		return v1.GreaterThan(v2)
	case "<<":
		return v1.LessThan(v2)
	case "=":
		return v1.Equal(v2)
	case ">=":
		return v1.GreaterThan(v2) || v1.Equal(v2)
	case "<=":
		return v1.LessThan(v2) || v1.Equal(v2)
	default:
		return true
	}
}
