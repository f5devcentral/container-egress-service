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
	SchemaVersion        string         `json:"schemaVersion"`
	ClusterName          string         `json:"clusterName"`
	MasterCluster        string         `json:"master_cluster"`
	IsSupportRouteDomain bool           `json:"isSupportRouteDomain"`
	Tenant               []TenantConfig `json:"tenant"`
	LogPool              LogPool        `json:"logPool"`
}

type TenantConfig struct {
	Name           string         `json:"name"`
	Namespaces     string         `json:"namespaces"`
	RouteDomain    RouteDomain    `json:"routeDomain"`
	Gwpool         Gwpool         `json:"gwPool"`
	VirtualService VirtualService `json:"virtualService"`
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

type LogPool struct {
	EnableRemoteLog bool     `json:"enableRemoteLog"`
	Template        string   `json:"template"`
	ServerAddresses []string `json:"serverAddresses"`
}

type BigIpAddressList struct {
	Addresses []BigIpAddresses `json:"addresses"`
}

type BigIpAddresses struct {
	Name string `json:"name"`
}

//Full body request struct
type (
	as3JSONWithArbKeys map[string]interface{}

	as3 as3JSONWithArbKeys

	as3ADC as3JSONWithArbKeys

	as3Tenant as3JSONWithArbKeys

	as3Application as3JSONWithArbKeys

	as3Declaration string

	poolName   string
	appName    string
	tenantName string
	tenant     map[appName][]poolName
)

type (
	protocol map[string][]string

	exsvcDate struct {
		name        string
		destPorts   protocol
		destAddress []string
	}
	ruleData struct {
		ty        string
		name      string
		namespace string
		action    string
		srcAddr   []string
		//ep name
		epName string
		exsvcs []*exsvcDate
	}
)

type changingRule struct {
	partition string
	exist     bool
	patchBody PatchBody
	value     interface{}
}
