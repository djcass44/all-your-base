package yumindex

import (
	"encoding/xml"
)

type Metadata struct {
	XMLName  xml.Name  `xml:"metadata"`
	Text     string    `xml:",chardata"`
	Packages string    `xml:"packages,attr"`
	Xmlns    string    `xml:"xmlns,attr"`
	Rpm      string    `xml:"rpm,attr"`
	Package  []Package `xml:"package"`
	// Source is a custom attribute that contains the original
	// repository address
	Source string `xml:"-"`
}

type Package struct {
	Text    string `xml:",chardata"`
	Type    string `xml:"type,attr"`
	Name    string `xml:"name"`
	Arch    string `xml:"arch"`
	Version struct {
		Text  string `xml:",chardata"`
		Epoch string `xml:"epoch,attr"`
		Rel   string `xml:"rel,attr"`
		Ver   string `xml:"ver,attr"`
	} `xml:"version"`
	Checksum struct {
		Text  string `xml:",chardata"`
		Pkgid string `xml:"pkgid,attr"`
		Type  string `xml:"type,attr"`
	} `xml:"checksum"`
	Summary     string `xml:"summary"`
	Description string `xml:"description"`
	Packager    string `xml:"packager"`
	URL         string `xml:"url"`
	Time        struct {
		Text  string `xml:",chardata"`
		Build string `xml:"build,attr"`
		File  string `xml:"file,attr"`
	} `xml:"time"`
	Size struct {
		Text      string `xml:",chardata"`
		Archive   string `xml:"archive,attr"`
		Installed string `xml:"installed,attr"`
		Package   string `xml:"package,attr"`
	} `xml:"size"`
	Location struct {
		Text string `xml:",chardata"`
		Href string `xml:"href,attr"`
	} `xml:"location"`
	Format struct {
		Text        string `xml:",chardata"`
		License     string `xml:"license"`
		Vendor      string `xml:"vendor"`
		Group       string `xml:"group"`
		Buildhost   string `xml:"buildhost"`
		Sourcerpm   string `xml:"sourcerpm"`
		HeaderRange struct {
			Text  string `xml:",chardata"`
			End   string `xml:"end,attr"`
			Start string `xml:"start,attr"`
		} `xml:"header-range"`
		Provides struct {
			Text  string    `xml:",chardata"`
			Entry EntryList `xml:"entry"`
		} `xml:"provides"`
		Requires struct {
			Text  string    `xml:",chardata"`
			Entry EntryList `xml:"entry"`
		} `xml:"requires"`
		File []struct {
			Text string `xml:",chardata"`
			Type string `xml:"type,attr"`
		} `xml:"file"`
		Conflicts struct {
			Text  string    `xml:",chardata"`
			Entry EntryList `xml:"entry"`
		} `xml:"conflicts"`
		Obsoletes struct {
			Text  string    `xml:",chardata"`
			Entry EntryList `xml:"entry"`
		} `xml:"obsoletes"`
		Suggests struct {
			Text  string    `xml:",chardata"`
			Entry EntryList `xml:"entry"`
		} `xml:"suggests"`
		Recommends struct {
			Text  string    `xml:",chardata"`
			Entry EntryList `xml:"entry"`
		} `xml:"recommends"`
		Supplements struct {
			Text  string `xml:",chardata"`
			Entry struct {
				Text string `xml:",chardata"`
				Name string `xml:"name,attr"`
			} `xml:"entry"`
		} `xml:"supplements"`
	} `xml:"format"`
}

type Entry struct {
	Text  string `xml:",chardata"`
	Name  string `xml:"name,attr"`
	Epoch string `xml:"epoch,attr"`
	Flags string `xml:"flags,attr"`
	Rel   string `xml:"rel,attr"`
	Ver   string `xml:"ver,attr"`
}

type EntryList []Entry

func (e EntryList) GetValues() []string {
	v := make([]string, len(e))
	for i := range e {
		v[i] = e[i].Name
	}
	return v
}
