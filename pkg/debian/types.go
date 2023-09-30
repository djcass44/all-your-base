package debian

type Package struct {
	Package      string
	Version      string
	Architecture string
	Depends      []string `delim:", "`
	Filename     string
	Sha256       string `control:"SHA256"`
}

type Index struct {
	packages []Package
	source   string
}

type PackageVersion struct {
	Names      []string
	Version    string
	Constraint string
}
