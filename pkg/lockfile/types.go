package lockfile

type Lock struct {
	Name            string             `json:"name"`
	LockfileVersion int                `json:"lockfileVersion"`
	Packages        map[string]Package `json:"packages"`
}

type Package struct {
	Name      string `json:"-"`
	Resolved  string `json:"resolved"`
	Integrity string `json:"integrity"`
}
