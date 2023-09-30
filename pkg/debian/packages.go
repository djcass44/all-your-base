package debian

import (
	"errors"
	"regexp"
	"strings"
)

var regexpParseVersion = regexp.MustCompile(`\((?P<constraint>\W{1,2})?(?P<version>.*)\)`)
var regexpName = regexp.MustCompile(`^[^([]+`)

// ParseVersion parses a debian version as used in the "Depends" section.
//
// https://www.debian.org/doc/debian-policy/ch-relationships.html
func ParseVersion(s string) (*PackageVersion, error) {
	matches := regexpName.FindStringSubmatch(s)
	if len(matches) == 0 {
		return nil, errors.New("unable to extract package names")
	}
	// extract the possible names
	names := strings.Split(matches[0], " | ")
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}
	// extract the version and constraint if they're present
	matches = regexpParseVersion.FindStringSubmatch(strings.TrimPrefix(s, matches[0]))
	var version string
	var constraint string
	if len(matches) >= 2 {
		version = strings.TrimSpace(matches[regexpParseVersion.SubexpIndex("version")])
		constraint = strings.TrimSpace(matches[regexpParseVersion.SubexpIndex("constraint")])
	}
	return &PackageVersion{
		Names:      names,
		Version:    version,
		Constraint: constraint,
	}, nil
}

func (p *Package) String() string {
	return p.Package + p.Version
}
