package lockfile

import (
	"fmt"
	v1 "github.com/djcass44/all-your-base/pkg/api/v1"
	"sort"
)

// Validate checks that the configuration file lines up
// with what we expect from the lockfile and vice versa
func (l *Lock) Validate(cfg v1.BuildSpec) error {
	// check that the krm packages are all in the lockfile
	for _, p := range cfg.Packages {
		for _, n := range p.Names {
			_, ok := l.Packages[n]
			if !ok {
				return fmt.Errorf("package not found in lock: %s", n)
			}
		}
	}
	// check that the krm files are all in the lockfile
	for _, f := range cfg.Files {
		_, ok := l.Packages[f.URI]
		if !ok {
			return fmt.Errorf("file not found in lock: %s", f.URI)
		}
	}

	// now we do the reverse

	for k, v := range l.Packages {
		if k == "" {
			continue
		}
		var found bool
		// check that the lock file are all present in the manifest
		if v.Type == v1.PackageFile {
			for _, f := range cfg.Files {
				if f.URI == k {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("file found in lock, but not manifest: %s", k)
			}
			continue
		}
		// check that the lock packages are all in the manifest
		for _, p := range cfg.Packages {
			for _, n := range p.Names {
				if n == k {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("package found in lock, but not manifest: %s", k)
		}
	}

	return nil
}

// SortedKeys returns package names
// sorted alphabetically.
func (l *Lock) SortedKeys() []string {
	pkgKeys := make([]string, 0)
	for k := range l.Packages {
		pkgKeys = append(pkgKeys, k)
	}
	sort.Strings(pkgKeys)
	return pkgKeys
}
