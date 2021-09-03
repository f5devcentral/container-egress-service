package as3

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"sigs.k8s.io/yaml"
	"github.com/tidwall/gjson"
)

func NewVirtualServer(nsConfig *As3Namespace) (vs VirtualServer, err error) {
	if strings.TrimSpace(nsConfig.VirtualService.Template) != "" {
		err = json.Unmarshal([]byte(nsConfig.VirtualService.Template), &vs)
		if err == nil{
		}
		return
	}
	vsPath := fmt.Sprintf("%s_svc_policy_%s", AS3PathPrefix(nsConfig), nsConfig.RouteDomain.Name)
	cluster := GetCluster()
	poolName := cluster + "_gw_pool"
	if !IsSupportRouteDomain(){
		//because only one vs
		vsPath = fmt.Sprintf("/Common/Shared/%s_svc_policy_rd", cluster)
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
	return fmt.Sprintf(pathProfix, nsConfig.Parttion, GetCluster())
}

func InitAs3Tenant(client *Client, filePath string,initialized bool) error{
	configData, err := ioutil.ReadFile(filePath+"/ces-conf.yaml")
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", filePath, err)
	}
	var as3Config As3Config
	err = yaml.Unmarshal(configData, &as3Config)
	if err != nil {
		return err
	}

	var as3 map[string]interface{}
	err = json.Unmarshal([]byte(as3Config.Base), &as3)
	if err != nil {
		return fmt.Errorf("init as3TenantTemplate error: " + err.Error())
	}

	as3Tenant, ok := as3[DeclarationKey].(map[string]interface{})[CommonKey].(map[string]interface{})
	if !ok {
		return fmt.Errorf("init as3TenantTemplate error: %v", err)
	}

	//store tenant in in sync.Map
	for _, nsconf :=range as3Config.Namespaces{
		if nsconf.Parttion == CommonKey{
			nsconf.RouteDomain = RouteDomain{
				Id: 0,
				Name: "0",
			}
		}
		registValue(nsconf.Name, nsconf)
	}

	//store cluster in sync.Map
	registValue(currentClusterKey, as3Config.ClusterName)
	registValue(isSupportRouteDomainKey, as3Config.IsSupportRouteDomain)

	svcRouteDomainPolicePath := fmt.Sprintf("/Common/Shared/%s_svc_policy_rd0", as3Config.ClusterName)
	if !as3Config.IsSupportRouteDomain{
		//because only one svc police
		svcRouteDomainPolicePath = fmt.Sprintf("/Common/Shared/%s_svc_policy_rd", as3Config.ClusterName)
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

	nsConfig := GetConfigNamespace(as3CommonPartitionKey)
	if nsConfig == nil{
		msg :=`
namespaces:
  ##common partiton config, init AS3 needs
  - name: "__common__"
    parttion: Common
    virtualService:
      template: ''
    gwPool:
      serverAddresses:
        - "192.168.10.1"
`
		return fmt.Errorf("No configured Common, please configured, eg: \n%s\n", msg)

	}
	commonTenant, err:= NewAs3Tenant(nsConfig, items)
	if err != nil{
		panic(err)
	}
	for k, v := range commonTenant {
		as3Tenant[k] = v
	}
	as3[DeclarationKey].(map[string]interface{})[CommonKey] = as3Tenant

	registValue(as3DefaultTemplateKey, as3Tenant)

	if !initialized {
		return client.Post(as3)
	}
	return nil
}

func GetDefaultTemplate()map[string]interface{}{
	v := getValue(as3DefaultTemplateKey)
	return v.(map[string]interface{})
}

func GetCluster() string{
	v := getValue(currentClusterKey)
	return v.(string)
}

func IsSupportRouteDomain() bool{
	v := getValue(isSupportRouteDomainKey)
	return v.(bool)
}

func IsNotFound(err error) bool {
	if strings.Contains(err.Error(), "status code 404") {
		return true
	}
	return false
}

func NewAs3Tenant(nsConfig *As3Namespace, patchBody []PatchItem) (map[string]interface{}, error) {
	vs, err := NewVirtualServer(nsConfig)
	if err != nil {
		return nil, err
	}

	cluster := GetCluster()
	vsPathName := cluster + "_outbound_vs"
	if !IsSupportRouteDomain(){
		//because only one vs
		vsPathName = fmt.Sprintf("%s_outbound_vs", cluster)
	}

	app := map[string]interface{}{
		ClassKey:    ClassApplication,
		TemplateKey: SharedValue,
		cluster + DenyAllRuleListName: FirewallRuleList{
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
			if !IsSupportRouteDomain(){
				use = Use{fmt.Sprintf("/Common/Shared/%s%s", cluster, DenyAllRuleListName)}
			}
			policy.Rules = append(policy.Rules, use)
			item.Value = policy
		}
		key := item.Path[index:]
		app[key] = item.Value
	}
	as3Tenant := map[string]interface{}{
		ClassKey:  ClassTenant,
		SharedKey: app,
	}
	return as3Tenant, nil
}


func GetConfigNamespace(namespace string) *As3Namespace {
	v := getValue(namespace)
	if v == nil{
		return nil
	}
	ns := v.(As3Namespace)
	if !IsSupportRouteDomain() {
		//not support rd, set Common, rd = 0
		ns.Parttion = "Common"
		ns.RouteDomain.Id = 0
		ns.RouteDomain.Name = "0"
	}
	return &ns
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
