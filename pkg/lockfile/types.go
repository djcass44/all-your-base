package lockfile

type Lock struct {
	Name            string             `json:"name"`
	LockfileVersion int                `json:"lockfileVersion"`
	Packages        map[string]Package `json:"packages"`
}

type Package struct {
	Name      string `json:"-"`
	Version   string `json:"version"`
	Resolved  string `json:"resolved"`
	Integrity string `json:"integrity"`
}
