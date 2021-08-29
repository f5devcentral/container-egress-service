package as3

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

var as3DefaultTemplate map[string]interface{}

var as3Config As3Config

func NewVirtualServer(nsConfig *As3Namespace, isNs bool) (vs VirtualServer, err error) {
	if strings.TrimSpace(nsConfig.VirtualService.Template) != "" {
		err = json.Unmarshal([]byte(nsConfig.VirtualService.Template), &vs)
		return
	}
	vsPath := fmt.Sprintf("%s_svc_policy_%s", AS3PathPrefix(nsConfig), nsConfig.RouteDomain.Name)
	poolName :=  as3Config.ClusterName + "_gw_pool"
	if !GetAs3Config().IsSupportRouteDomain{
		//because only one vs
		vsPath = "/Common/Shared/k8s_svc_policy_rd"
		poolName = "k8s_gw_pool"
	}
	vs = VirtualServer{
		Layer4:                 "any",
		TranslateServerAddress: false,
		TranslateServerPort:    false,
		VirtualAddresses:       []string{"0.0.0.0"},
		PolicyFirewallEnforced: Use{
			vsPath,
		},
		SecurityLogProfiles: []Use{},
		VirtualPort:         0,
		Snat:                "auto",
		Class:               ClassVirtualServerL4,
		Pool:                poolName,
	}
	return
}

func NewPoll(serverAddresses []string) Pool {
	return Pool{
		Class: ClassPoll,
		Members: []Member{
			{
				ServerAddresses: serverAddresses,
				Enable:           true,
				ServicePort:     0,
			},
		},
		Monitors: []Monitor{
			{
				Bigip: "/Common/gateway_icmp",
			},
		},
	}
}

func AS3PathPrefix(nsConfig *As3Namespace) string {
	return fmt.Sprintf(pathProfix, nsConfig.Parttion, as3Config.ClusterName)
}

func InitAs3Tenant(template string, client *Client, initialized bool) error{
	var as3 map[string]interface{}
	err := json.Unmarshal([]byte(template), &as3)
	if err != nil {
		return fmt.Errorf("init as3TenantTemplate error: " + err.Error())
	}

	as3Tenant, ok := as3["declaration"].(map[string]interface{})["Common"].(map[string]interface{})
	if !ok {
		panic("init as3TenantTemplate error: " + err.Error())
	}

	var nsConfig *As3Namespace
	for _, nsconf :=range as3Config.Namespaces{
		if nsconf.Parttion == "Common"{
			nsConfig = &nsconf
			break
		}
	}
	if nsConfig == nil{
		return fmt.Errorf("failed to get Common partition data, please set in conf.yaml")
	}

	svcRouteDomainPolicePath := fmt.Sprintf("/Common/Shared/%s_svc_policy_%s", AS3PathPrefix(nsConfig), nsConfig.RouteDomain.Name)
	if !as3Config.IsSupportRouteDomain{
		//because only one svc police
		svcRouteDomainPolicePath = "/Common/Shared/k8s_svc_policy_rd"
	}

	items := []PatchItem{
		{
			Path: svcRouteDomainPolicePath,
			Value: FirewallPolicy{
				Class: ClassFirewallPolicy,
				Rules: []Use{},
			},
		},
	}
	commonTenant, err:= NewAs3Tenant(nsConfig, items, false)
	if err != nil{
		panic(err)
	}
	for k, v := range commonTenant {
		as3Tenant[k] = v
	}
	as3["declaration"].(map[string]interface{})["Common"] = as3Tenant
	as3DefaultTemplate = as3Tenant
	if !initialized {
		return client.Post(as3)
	}
	return nil
}

func GetAs3Tenant() (string, map[string]interface{}) {
	adc, _ := json.Marshal(as3DefaultTemplate)
	return string(adc), as3DefaultTemplate
}


func GetVsName(){

}

func IsNotFound(err error) bool {
	if strings.Contains(err.Error(), "status code 404") {
		return true
	}
	return false
}

func NewAs3Tenant(nsConfig *As3Namespace, patchBody []PatchItem, isNs bool) (map[string]interface{}, error) {
	vs, err := NewVirtualServer(nsConfig, isNs)
	if err != nil {
		return nil, err
	}

	vsPathName := as3Config.ClusterName + "_outbound_vs"
	if !GetAs3Config().IsSupportRouteDomain{
		//because only one vs
		vsPathName = "k8s_outbound_vs"
	}

	app := map[string]interface{}{
		"class":    "Application",
		"template": "shared",
		as3Config.ClusterName + DenyAllRuleListName: FirewallRuleList{
			Class: ClassFirewallRuleList,
			//Deny all by default
			Rules: []FirewallRule{
				{
					Protocol:    "any",
					Name:        DenyAllRuleName,
					Destination: FirewallDestination{},
					Source:      FirewallSource{},
					Action:      "drop",
				},
			},
		},
		vs.Pool:     NewPoll(nsConfig.Gwpool.ServerAddresses),
		vsPathName:  vs,
	}

	for _, item := range patchBody {
		index := strings.LastIndex(item.Path, "/") + 1
		// add deny_all in policyList'uses
		if policy, ok := item.Value.(FirewallPolicy); ok && strings.Contains(item.Path, "_svc_policy_") {
			use := Use{fmt.Sprintf("%s%s", AS3PathPrefix(nsConfig), DenyAllRuleListName)}
			if !as3Config.IsSupportRouteDomain{
				use = Use{fmt.Sprintf("/Common/Shared/k8s%s", DenyAllRuleListName)}
			}
			policy.Rules = append(policy.Rules, use)
			item.Value = policy
		}
		key := item.Path[index:]
		app[key] = item.Value
	}
	as3Tenant := map[string]interface{}{
		"Shared": app,
		"class":  "Tenant",
	}
	return as3Tenant, nil
}

func GetAs3Config() As3Config {
	return as3Config
}

func SetAs3Config(c As3Config) {
	as3Config = c
}

func GetConfigNamespace(namespace string) *As3Namespace {
	for _, c := range as3Config.Namespaces {
		if c.Name == namespace {
			if !as3Config.IsSupportRouteDomain {
				//not support rd, set Common, rd = 0
				c.Parttion = "Common"
				c.RouteDomain.Id = 0
				c.RouteDomain.Name = "0"
			}
			return &c
		}
	}
	return nil
}

func JudgeSelectedUpdate(adc string, items []PatchItem, isDelete bool) (newItems []PatchItem) {
	for _, item := range items {
		path := strings.ReplaceAll(item.Path[1:], "/", ".")
		if gjson.Get(adc, path).Exists() {
			if !isDelete {
				obj1 := gjson.Get(adc, path).String()
				obj2 := item.Value
				if isObj1EqualObj2(obj1, obj2) {
					continue
				} else {
					item.Op = OpReplace
				}

			} else {
				item.Op = OpRemove
				item.Value = nil
			}
			newItems = append(newItems, item)
		} else {
			if !isDelete {
				item.Op = OpAdd
				newItems = append(newItems, item)
			}
		}
	}
	return
}

func isObj1EqualObj2(adcJson, detal interface{}) bool {
	switch detal.(type) {
	case FirewallRuleList:
		var v1 FirewallRuleList
		err := json.Unmarshal([]byte(adcJson.(string)), &v1)
		if err != nil {
			return false
		}
		v2 := detal.(FirewallRuleList)
		if v2.Class != v1.Class {
			return false
		}
		for _, rule2 := range v2.Rules {
			isFind := false
			for _, rule1 := range v1.Rules {
				if rule2.Name == rule1.Name {
					isFind = true
					if !isObj1EqualObj2(rule1, rule2) {
						return false
					}
					break
				}
			}
			if !isFind{
				return false
			}
		}
		return true
	case FirewallRule:
		v1, ok := adcJson.(FirewallRule)
		if !ok {
			return false
		}
		v2 := detal.(FirewallRule)
		if v2.Name != v1.Name {
			return false
		}

		if v2.Action != v1.Action {
			return false
		}

		for _, use2 := range v2.Destination.AddressLists {
			isFind := false
			for _, use1 := range v1.Destination.AddressLists {
				if use2.Use == use1.Use {
					isFind = true
					break
				}
			}
			if !isFind {
				return false
			}
		}

		for _, use2 := range v2.Destination.PortLists {
			isFind := false
			for _, use1 := range v1.Destination.PortLists {
				if use2.Use == use1.Use {
					isFind = true
					break
				}
			}
			if !isFind {
				return false
			}
		}
	case FirewallPortList:
		var v1 FirewallPortList
		err := json.Unmarshal([]byte(adcJson.(string)), &v1)
		if err != nil {
			return false
		}
		v2 := detal.(FirewallPortList)
		if v2.Class != v1.Class {
			return false
		}
		for _, port2 := range v2.Ports {
			isFind := false
			for _, port1 := range v1.Ports {
				if port2 == port1 {
					isFind = true
					break
				}
			}
			if !isFind {
				return false
			}
		}
	case FirewallAddressList:
		var v1 FirewallAddressList
		err := json.Unmarshal([]byte(adcJson.(string)), &v1)
		if err != nil {
			return false
		}
		v2 := detal.(FirewallAddressList)
		if v2.Class != v1.Class {
			return false
		}
		for _, addr2 := range v2.Addresses {
			isFind := false
			for _, addr1 := range v1.Addresses {
				if addr2 == addr1 {
					isFind = true
					break
				}
			}
			if !isFind {
				return false
			}
		}
	}
	return true
}
