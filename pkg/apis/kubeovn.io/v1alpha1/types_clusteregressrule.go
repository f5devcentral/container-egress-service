package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
// ClusterEgressRule, is the spec for an ClusterEgressRule, resource
type ClusterEgressRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ClusterEgressRuleSpec `json:"spec"`
	Status ClusterEgressRuleStatus `json:"status"`
}

// ClusterEgressRuleSpec is the spec for an ClusterEgressRule resource
type ClusterEgressRuleSpec struct {
	Action           string   `json:"action"`
	ExternalServices []string `json:"externalServices"`
}

type ClusterEgressRuleStatus struct {
	Phase ClusterEgressRulePhase `json:"phase,omitempty"`
}

type ClusterEgressRulePhase string

const (
	ClusterEgressRuleSuccess ClusterEgressRulePhase = "Success"
	ClusterEgressRuleSyncing ClusterEgressRulePhase = "Syncing"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterEgressRuleList is a list of ClusterEgressRule resources
type ClusterEgressRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ClusterEgressRule `json:"items"`
}

