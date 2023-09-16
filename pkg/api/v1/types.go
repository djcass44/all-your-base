package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type BuildSpec struct {
	From     string    `json:"from,omitempty"`
	Packages []Package `json:"packages,omitempty"`
}

type Package struct {
	URL string `json:"url,omitempty"`
}

type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BuildSpec `json:"spec"`
}
