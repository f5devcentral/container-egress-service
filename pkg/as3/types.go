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
	Fqdns     []string `json:"fqdns,omitempty"`
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
	IRule       *IRule              `json:"iRule,omitempty"`
	Destination FirewallDestination `json:"destination,omitempty"`

	Source         FirewallSource `json:"source,omitempty"`
	Action         string         `json:"action,omitempty"`
	LoggingEnabled bool           `json:"loggingEnabled,omitempty"`
}

type IRule struct {
	Bigip string `json:"bigip,omitempty"`
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
	Layer4                 string `json:"layer4"`
	TranslateServerAddress bool   `json:"translateServerAddress"`
	TranslateServerPort    bool   `json:"translateServerPort"`
	VirtualAddresses       []Use  `json:"virtualAddresses"`
	PolicyFirewallEnforced Use    `json:"policyFirewallEnforced"`
	SecurityLogProfiles    []Use  `json:"securityLogProfiles,omitempty"`
	VirtualPort            int    `json:"virtualPort"`
	Snat                   string `json:"snat"`
	PolicyNAT              Use    `json:"policyNAT"`
	Class                  string `json:"class"`
	Pool                   string `json:"pool"`
}

// ARP
type VirtualServerVa struct {
	Class          string `json:"class"`
	VirtualAddress string `json:"virtualAddress"`
	IcmpEcho       string `json:"icmpEcho"`
	ArpEnabled     bool   `json:"arpEnabled"`
}

// viper
type (
	As3Config struct {
		SchemaVersion        string         `mapstructure:"schemaVersion"`
		ClusterName          string         `mapstructure:"clusterName"`
		MasterCluster        string         `mapstructure:"masterCluster"`
		IsSupportRouteDomain bool           `mapstructure:"isSupportRouteDomain"`
		IRule                []string       `mapstructure:"iRule"`
		Tenant               []TenantConfig `mapstructure:"tenant"`
		ExternalIPAddresses  []string       `mapstructure:"externalIPAddresses"`
		LogPool              LogPool        `mapstructure:"logPool"`
	}

	LogPool struct {
		//Whether to configure logging profile
		LoggingEnabled bool `mapstructure:"loggingEnabled"`
		//Whether to open remote log
		EnableRemoteLog bool     `mapstructure:"enableRemoteLog"`
		HealthMonitor   string   `mapstructure:"healthMonitor"`
		Template        string   `mapstructure:"template"`
		ServerAddresses []string `mapstructure:"serverAddresses"`
	}

	TenantConfig struct {
		Name           string         `mapstructure:"name"`
		Namespaces     string         `mapstructure:"namespaces"`
		RouteDomain    RouteDomain    `mapstructure:"routeDomain"`
		Gwpool         Gwpool         `mapstructure:"gwPool"`
		VirtualService VirtualService `mapstructure:"virtualService"`
	}

	RouteDomain struct {
		Id        int    `mapstructure:"id,omitempty"`
		Name      string `mapstructure:"name,omitempty"`
		Partition string `mapstructure:"partition,omitempty"`
	}

	Gwpool struct {
		ServerAddresses []string `mapstructure:"serverAddresses"`
	}

	VirtualService struct {
		//Custom vs structure，if "", use Common vs value
		Template         string           `mapstructure:"template"`
		VirtualAddresses VirtualAddresses `mapstructure:"virtualAddresses"`
	}

	VirtualAddresses struct {
		VirtualAddress string `mapstructure:"virtualAddress"`
		IcmpEcho       string `mapstructure:"icmpEcho"`
		ArpEnabled     bool   `mapstructure:"arpEnabled"`
		template       string `mapstructure:"template"`
	}
)

// BIG-IP
type (
	BigIpAddressList struct {
		Addresses []BigIpAddresses `json:"addresses"`
	}
	BigIpAddresses struct {
		Name string `json:"name"`
	}
)

// Full body request struct
type (
	as3JSONWithArbKeys map[string]interface{}

	as3 as3JSONWithArbKeys

	as3ADC as3JSONWithArbKeys

	as3Tenant as3JSONWithArbKeys

	as3Application as3JSONWithArbKeys

	as3Declaration string
)

type (
	portIrule struct {
		protocol string
		irule    string
		ports    []string
	}

	//protocol map[string]portIrule

	exsvcDate struct {
		name        string
		destPorts   map[string]portIrule
		destAddress []string
	}
	ruleData struct {
		ty        string
		name      string
		namespace string
		action    string
		logging   bool
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
