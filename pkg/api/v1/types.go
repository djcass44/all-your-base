package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type PackageType string

const (
	PackageAlpine PackageType = "Alpine"
	PackageDebian PackageType = "Debian"
	PackageRPM    PackageType = "RPM"
)

type BuildSpec struct {
	From         string                  `json:"from,omitempty"`
	Packages     []Package               `json:"packages,omitempty"`
	Repositories map[string][]Repository `json:"repositories,omitempty"`
	Files        []File                  `json:"files,omitempty"`
}

type Repository struct {
	URL string `json:"url"`
}

type Package struct {
	Type PackageType `json:"type"`
	Name string      `json:"name,omitempty"`
	URL  string      `json:"url,omitempty"`
}

type File struct {
	URI  string `json:"uri"`
	Path string `json:"path"`
}

type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BuildSpec `json:"spec"`
}
