package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MedusaSpec defines the desired state of Medusa
type MedusaSpec struct {
	Backend Backend `json:"backend,omitempty"`
	Admin   Admin   `json:"admin,omitempty"`
}

type Common struct {
	Image  string `json:"image,omitempty"`
	Domain string `json:"domain,omitempty"`
}

type Admin struct {
	Common `json:",inline"`
}

type Backend struct {
	Common   `json:",inline"`
	RedisURL string `json:"redisURL,omitempty"`
}

// MedusaStatus defines the observed state of Medusa
type MedusaStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Medusa is the Schema for the medusas API
type Medusa struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MedusaSpec   `json:"spec,omitempty"`
	Status MedusaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MedusaList contains a list of Medusa
type MedusaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Medusa `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Medusa{}, &MedusaList{})
}
