package yumrepo

func (d *RepoData) PrimaryXML() string {
	for _, i := range d.Data {
		if i.Type == "primary" {
			return i.Location.Href
		}
	}
	return ""
}
