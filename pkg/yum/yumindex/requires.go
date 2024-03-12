package yumindex

import (
	"context"
	"github.com/go-logr/logr"
	"golang.org/x/exp/maps"
	"slices"
)

func (m *Metadata) GetProviders(ctx context.Context, requires []string) []Package {
	log := logr.FromContextOrDiscard(ctx)
	log.V(1).Info("checking for packages", "requires", requires)

	// collect a list of unique package matches
	matches := map[string]Package{}

	for _, pkg := range m.Package {
		log.V(4).Info("checking package", "name", pkg.Name, "provides", pkg.Format.Provides)
		for _, e := range pkg.Format.Provides.Entry {
			// bunch of them are empty so just skip
			// them
			if e.Name == "" {
				continue
			}
			if slices.Contains(requires, e.Name) {
				log.V(2).Info("found matching package", "entry", e.Name)
				matches[pkg.Name] = pkg
			}
		}
	}

	return maps.Values(matches)
}
