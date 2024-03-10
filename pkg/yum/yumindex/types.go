package yumindex

import "encoding/xml"

type Metadata struct {
	XMLName      xml.Name  `xml:"metadata"`
	Packages     int64     `xml:"packages,attr"`
	PackagesList []Package `xml:"package"`
	// Source is a custom attribute that contains the original
	// repository address
	Source string `xml:"-"`
}

type Package struct {
	XMLName     xml.Name `xml:"package"`
	Type        string   `xml:"type,attr"`
	Name        string   `xml:"name"`
	Arch        string   `xml:"arch"`
	Version     Version  `xml:"version"`
	Checksum    Checksum `xml:"checksum"`
	Summary     string   `xml:"summary"`
	Description string   `xml:"description"`
	Packager    string   `xml:"packager"`
	URL         string   `xml:"url"`
	Time        Time     `xml:"time"`
	Size        Size     `xml:"size"`
	Location    Location `xml:"location"`
	Format      Format   `xml:"format"`
	Files       []string `xml:"file"`
}

type Location struct {
	Href string `xml:"href,attr"`
}

type Version struct {
	Epoch int64  `xml:"epoch,attr"`
	Rel   string `xml:"rel,attr"`
	Ver   string `xml:"ver,attr"`
}

type Checksum struct {
	Pkgid string `xml:"pkgid,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type Time struct {
	Build int64 `xml:"build,attr"`
	File  int64 `xml:"file,attr"`
}

type Size struct {
	Archive   int64 `xml:"archive,attr"`
	Installed int64 `xml:"installed,attr"`
	Package   int64 `xml:"package,attr"`
}

type Format struct {
	XMLName     xml.Name `xml:"format"`
	License     string   `xml:"rpm:license"`
	Vendor      string   `xml:"rpm:vendor"`
	Group       string   `xml:"rpm:group"`
	Buildhost   string   `xml:"rpm:buildhost"`
	Sourcerpm   string   `xml:"rpm:sourcerpm"`
	HeaderRange string   `xml:"rpm:header-range"`
	Provides    Provides `xml:"rpm:provides"`
	Requires    Requires `xml:"rpm:requires"`
}

type Provides struct {
	Entry []Entry `xml:"rpm:entry"`
}

type Requires struct {
	Entry []Entry `xml:"rpm:entry"`
}

type Entry struct {
	Epoch int64  `xml:"epoch,attr"`
	Flags string `xml:"flags,attr"`
	Name  string `xml:"name,attr"`
	Rel   string `xml:"rel,attr"`
	Ver   string `xml:"ver,attr"`
}
