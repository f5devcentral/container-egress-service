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

package as3

const (
	// patch operations
	OpAdd     = "add"
	OpRemove  = "remove"
	OpReplace = "replace"
)

const (
	//ADC class
	ClassApplication = "Application"
	ClassTenant      = "Tenant"

	// AS3 classes
	ClassFirewallAddressList = "Firewall_Address_List"
	ClassFirewallPortList    = "Firewall_Port_List"
	ClassFirewallRuleList    = "Firewall_Rule_List"
	ClassFirewallPolicy      = "Firewall_Policy"
	ClassVirtualServerL4     = "Service_L4"
	ClassPoll                = "Pool"
)

const (
	ClassKey                  = "class"
	TemplateKey               = "template"
	SharedKey                 = "Shared"
	CommonKey                 = "Common"
	DeclarationKey            = "declaration"
	EnforcedPolicyKey         = "enforcedPolicy"
	FwEnforcedPolicyKey       = "fwEnforcedPolicy"
	DefaultRouteDomainKey     = "defaultRouteDomain"
	PolicyFirewallEnforcedKey = "policyFirewallEnforced"

	SharedValue = "shared"
	TenantValue = "Tenant"
)

const (
	RuleTypeLabel = "cpaas.io/ruleType"

	RuleTypeGlobal    = "global"
	RuleTypeNamespace = "namespace"
	RuleTypeService   = "service"
)

const (
	DenyAllRuleListName = "_svc_deny_all_rule_list"
	DenyAllRuleName     = "deny_all_rule"
)

const (
	ClusterSvcExtNamespace = "kube-system"
)

//eg: Common/Shared/k8s
const pathProfix = "/%s/Shared/%s"

// PatchItem represents a JSON patch item
type PatchItem struct {
	Op    string      `json:"op,omitempty"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// PatchItem represents a JSON patch body
type PatchBody []PatchItem

// FirewallAddressList represents a firewall address list
type FirewallAddressList struct {
	Class     string   `json:"class,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
}

// FirewallPortList represents a firewall port list
type FirewallPortList struct {
	Class string   `json:"class,omitempty"`
	Ports []string `json:"ports,omitempty"`
}

// FirewallRuleList represents a firewall rule list
type FirewallRuleList struct {
	Class string         `json:"class,omitempty"`
	Rules []FirewallRule `json:"rules,omitempty"`
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	Protocol    string              `json:"protocol,omitempty"`
	Name        string              `json:"name,omitempty"`
	Destination FirewallDestination `json:"destination,omitempty"`

	Source FirewallSource `json:"source,omitempty"`
	Action string         `json:"action,omitempty"`
}

// FirewallRule represents a firewall destination
type FirewallDestination struct {
	AddressLists []Use `json:"addressLists,omitempty"`
	PortLists    []Use `json:"portLists,omitempty"`
}

// FirewallRule represents a firewall source
type FirewallSource FirewallDestination

// FirewallRule represents an AS3 use declaration
type Use struct {
	Use string `json:"use"`
}

type FirewallPolicy struct {
	Class string `json:"class,omitempty"`
	Rules []Use  `json:"rules"`
}

type F5ApiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Pool struct {
	Class    string    `json:"class"`
	Members  []Member  `json:"members"`
	Monitors []Monitor `json:"monitors"`
}

type Member struct {
	ServerAddresses []string `json:"serverAddresses"`
	Enable          bool     `json:"enable"`
	ServicePort     int      `json:"servicePort"`
	//BigIp           string   `json:"bigip"`
}
type Monitor struct {
	Bigip string `json:"bigip"`
}

type VirtualServer struct {
	Layer4                 string   `json:"layer4"`
	TranslateServerAddress bool     `json:"translateServerAddress"`
	TranslateServerPort    bool     `json:"translateServerPort"`
	VirtualAddresses       []string `json:"virtualAddresses"`
	PolicyFirewallEnforced Use      `json:"policyFirewallEnforced"`
	SecurityLogProfiles    []Use    `json:"securityLogProfiles,omitempty"`
	VirtualPort            int      `json:"virtualPort"`
	Snat                   string   `json:"snat"`
	Class                  string   `json:"class"`
	Pool                   string   `json:"pool"`
}

type As3Config struct {
	Base                 string         `json:"base"`
	ClusterName          string         `json:"clusterName"`
	IsSupportRouteDomain bool           `json:"isSupportRouteDomain"`
	Namespaces           []As3Namespace `json:"namespaces"`
}

type As3Namespace struct {
	Name           string         `json:"name"`
	Parttion       string         `json:"parttion"`
	RouteDomain    RouteDomain    `json:"routeDomain"`
	VirtualService VirtualService `json:"virtualService"`
	Gwpool         Gwpool         `json:"gwPool"`
}

type RouteDomain struct {
	Id               int    `json:"id,omitempty"`
	Name             string `json:"name,omitempty"`
	Partition        string `json:"partition,omitempty"`
	FwEnforcedPolicy string `json:"fwEnforcedPolicy,omitempty"`
}

type VirtualService struct {
	//Custom vs structure，if "", use Common vs value
	Template string `json:"template"`
}

type Gwpool struct {
	ServerAddresses []string `json:"serverAddresses"`
}

type BigIpAddressList struct {
	Addresses []BigIpAddresses `json:"addresses"`
}

type BigIpAddresses struct {
	Name string `json:"name"`
}
