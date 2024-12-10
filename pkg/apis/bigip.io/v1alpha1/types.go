package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalIPRule is a specification for an ExternalIPRule resource
type ExternalIPRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ExternalIPRuleSpec `json:"spec"`
}

// DestinationMatch is a specification for an ExternalIPRule match
type DestinationMatch struct {
	Name      string                `json:"name"`
	Ports     DestinationMatchPorts `json:"destinationMatchPorts,omitempty"`
	Addresses []string              `json:"addresses,omitempty"`
}

type DestinationMatchPorts struct {
	Protocol string   `json:"protocol,omitempty"`
	Ports    []string `json:"ports,omitempty"`
}

// ExternalIPRuleSpec is the spec for a ExternalIPRule resource
type ExternalIPRuleSpec struct {
	Priority          uint32           `json:"priority"`
	ExternalAddresses []string         `json:"externalAddresses"`
	DestinationMatch  DestinationMatch `json:"destinationMatch,omitempty"`
	Services          []string         `json:"services"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalIPRuleList is a list of ExternalIPRule resources
type ExternalIPRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ExternalIPRule `json:"items"`
}
