package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalService is a specification for an ExternalService resource
type ExternalService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ExternalServiceSpec `json:"spec"`
}

// ExternalService is a specification for an ExternalService port
type ExternalServicePort struct {
	Name      string `json:"name"`
	Protocol  string `json:"protocol"`
	Port      string `json:"port"`
	Bandwidth string `json:"bandwidth,omitempty"`
}

// ExternalServiceSpec is the spec for a ExternalService resource
type ExternalServiceSpec struct {
	Addresses []string              `json:"addresses"`
	Ports     []ExternalServicePort `json:"ports,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalServiceList is a list of ExternalService resources
type ExternalServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ExternalService `json:"items"`
}
