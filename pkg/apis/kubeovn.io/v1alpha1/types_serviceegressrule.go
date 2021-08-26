/*
Copyright 2021 The Kube-OVN AS3 Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"


// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceEgressRule is a specification for an F5TrafficControlRule resource
type ServiceEgressRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ServiceEgressRuleSpec `json:"spec"`
	
	Status ServiceEgressRuleStatus `json:"status"`
}

// ServiceEgressRuleSpec is the spec for an F5TrafficControlRule resource
type ServiceEgressRuleSpec struct {
	Action           string   `json:"action"`
	Service         string    `json:"service"`
	ExternalServices []string `json:"externalServices"`
}


type ServiceEgressRuleStatus struct {
	Phase ServiceEgressRulePhase `json:"phase,omitempty"`
}

type ServiceEgressRulePhase string

const (
	ServiceEgressRuleSuccess ServiceEgressRulePhase = "Success"
	ServiceEgressRuleSyncing ServiceEgressRulePhase = "Syncing"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceEgressRuleList is a list of ServiceEgressRule resources
type ServiceEgressRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ServiceEgressRule `json:"items"`
}
