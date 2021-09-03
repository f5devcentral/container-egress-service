package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceEgressRule is the spec for an NamespaceEgressRule resource
type NamespaceEgressRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NamespaceEgressRuleSpec   `json:"spec"`
	Status NamespaceEgressRuleStatus `json:"status"`
}

// NamespaceEgressRuleSpec is the spec for an NamespaceEgressRule resource
type NamespaceEgressRuleSpec struct {
	Action           string   `json:"action"`
	//Subnet           string   `json:"subnet"`
	ExternalServices []string `json:"externalServices"`
}

type NamespaceEgressRuleStatus struct {
	Phase NamespaceEgressRulePhase `json:"phase,omitempty"`
}

type NamespaceEgressRulePhase string

const (
	NamespaceEgressRuleSuccess NamespaceEgressRulePhase = "Success"
	NamespaceEgressRuleSyncing NamespaceEgressRulePhase = "Syncing"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceEgressRuleList is a list of NamespaceEgressRule resources
type NamespaceEgressRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NamespaceEgressRule `json:"items"`
}
