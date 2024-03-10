package yumrepo

import "encoding/xml"

type RepoData struct {
	XMLName  xml.Name `xml:"repomd"`
	Revision int64    `xml:"revision"`
	Data     []Data   `xml:"data"`
}

type Data struct {
	Type            string   `xml:"type,attr"`
	Checksum        Checksum `xml:"checksum"`
	OpenChecksum    Checksum `xml:"open-checksum"`
	Location        Location `xml:"location"`
	Timestamp       int64    `xml:"timestamp"`
	Size            int64    `xml:"size"`
	OpenSize        int64    `xml:"open-size"`
	DatabaseVersion *int64   `xml:"database_version,omitempty"`
}

type Checksum struct {
	Type  string `xml:"type,attr"`
	Value string `xml:"text"`
}

type Location struct {
	Href string `xml:"href,attr"`
}
