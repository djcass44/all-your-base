package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type PackageType string

const (
	PackageAlpine PackageType = "Alpine"
	PackageDebian PackageType = "Debian"
	PackageRPM    PackageType = "RPM"
	PackageFile   PackageType = "File"
	PackageOCI    PackageType = "OCI"
)

type BuildSpec struct {
	From         string                  `json:"from,omitempty"`
	Packages     []Package               `json:"packages,omitempty"`
	Repositories map[string][]Repository `json:"repositories,omitempty"`
	Files        []File                  `json:"files,omitempty"`
	Links        []Link                  `json:"links,omitempty"`
	Env          []EnvVar                `json:"env,omitempty"`
}

type Repository struct {
	URL string `json:"url"`
}

type Package struct {
	Type  PackageType `json:"type"`
	Names []string    `json:"names"`
}

type File struct {
	URI        string `json:"uri"`
	Path       string `json:"path"`
	Executable bool   `json:"executable"`
	SubPath    string `json:"subPath"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BuildSpec `json:"spec"`
}
