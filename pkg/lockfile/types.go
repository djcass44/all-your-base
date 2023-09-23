package lockfile

import v1 "github.com/djcass44/all-your-base/pkg/api/v1"

type Lock struct {
	Name            string             `json:"name"`
	LockfileVersion int                `json:"lockfileVersion"`
	Packages        map[string]Package `json:"packages"`
}

type Package struct {
	Name      string         `json:"-"`
	Type      v1.PackageType `json:"type"`
	Version   string         `json:"version"`
	Resolved  string         `json:"resolved"`
	Integrity string         `json:"integrity"`
}
