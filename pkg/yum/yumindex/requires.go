package yumindex

import (
	"context"
	"github.com/go-logr/logr"
	"golang.org/x/exp/maps"
)

func (m *Metadata) GetProviders(ctx context.Context, requires []string, existingPackages map[string]bool) []Package {
	log := logr.FromContextOrDiscard(ctx)
	log.V(3).Info("checking for packages", "requires", requires)

	// collect a list of unique package matches
	matches := map[string]Package{}

	for _, r := range requires {
		packages := m.GetPackageAndDependencies(ctx, r, existingPackages)
		for _, p := range packages {
			matches[p.Name] = p
		}
	}

	return maps.Values(matches)
}

func (m *Metadata) GetPackageAndDependencies(ctx context.Context, pkg string, existingPackages map[string]bool) []Package {
	log := logr.FromContextOrDiscard(ctx)
	log.V(3).Info("fetching package and dependencies", "pkg", pkg)

	if existingPackages == nil {
		existingPackages = map[string]bool{}
	}

	if existingPackages[pkg] != false {
		return nil
	}

	// collect a list of unique package matches
	matches := map[string]Package{}

	for _, p := range m.Package {
		for _, e := range p.Format.Provides.Entry {
			// bunch of them are empty so just skip
			// them
			if e.Name == "" {
				continue
			}
			if e.Name == pkg {
				log.V(4).Info("found matching package", "entry", e.Name)
				matches[pkg] = p
				existingPackages[pkg] = true

				// collect dependencies
				for _, r := range p.Format.Requires.Entry.GetValues() {
					packages := m.GetPackageAndDependencies(ctx, r, existingPackages)
					for _, p := range packages {
						matches[p.Name] = p
						existingPackages[p.Name] = true
					}
				}
			}
		}
	}

	return maps.Values(matches)
}
