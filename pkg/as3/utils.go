package as3

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	snat "github.com/kubeovn/ces-controller/pkg/apis/bigip.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// other partition set clusterEgressList is nil
// The purpose is not to set the global policy
type as3Post struct {
	serviceEgressList   *v1alpha1.ServiceEgressRuleList
	namespaceEgressList *v1alpha1.NamespaceEgressRuleList
	clusterEgressList   *v1alpha1.ClusterEgressRuleList
	externalServiceList *v1alpha1.ExternalServiceList
	externalIPRuleList  *snat.ExternalIPRuleList
	endpointList        *corev1.EndpointsList
	namespaceList       *corev1.NamespaceList
	tenantConfig        *TenantConfig
}

func newAs3Post(serviceEgressList *v1alpha1.ServiceEgressRuleList, namespaceEgressList *v1alpha1.NamespaceEgressRuleList,
	clusterEgressList *v1alpha1.ClusterEgressRuleList, externalServiceList *v1alpha1.ExternalServiceList,
	externalIPRuleList *snat.ExternalIPRuleList,
	endpointList *corev1.EndpointsList, namespaceList *corev1.NamespaceList, tenantConfig *TenantConfig) *as3Post {
	//init default value, make sure not nil pointer
	ac := as3Post{
		serviceEgressList:   &v1alpha1.ServiceEgressRuleList{},
		namespaceEgressList: &v1alpha1.NamespaceEgressRuleList{},
		clusterEgressList:   &v1alpha1.ClusterEgressRuleList{},
		externalServiceList: &v1alpha1.ExternalServiceList{},
		externalIPRuleList:  &snat.ExternalIPRuleList{},
		endpointList:        &corev1.EndpointsList{},
		namespaceList:       &corev1.NamespaceList{},
		tenantConfig:        tenantConfig,
	}

	if serviceEgressList != nil {
		ac.serviceEgressList = serviceEgressList
	}
	if namespaceEgressList != nil {
		ac.namespaceEgressList = namespaceEgressList
	}
	if clusterEgressList != nil {
		ac.clusterEgressList = clusterEgressList
	}
	if externalServiceList != nil {
		ac.externalServiceList = externalServiceList
	}
	if endpointList != nil {
		ac.endpointList = endpointList
	}
	if namespaceList != nil {
		ac.namespaceList = namespaceList
	}
	if externalIPRuleList != nil {
		ac.externalIPRuleList = externalIPRuleList
	}
	return &ac
}

func initDefaultAS3() as3 {
	as3 := as3{}
	as3["$schema"] = "https://raw.githubusercontent.com/F5Networks/f5-appsvcs-extension/master/schema/latest/as3-schema.json"
	as3[ClassKey] = classAS3
	as3.initDefault()
	return as3
}

func (ac as3) initDefault() {
	adc := as3ADC{}
	adc.initDefault(DefaultPartition)
	adc[ClassKey] = ClassADC
	adc["id"] = "k8s-ces-controller"
	adc["schemaVersion"] = getSchemaVersion()
	adc["updateMode"] = "selective"
	ac[DeclarationKey] = adc
}

func newAs3Obj(partition string, shareApplication interface{}) interface{} {
	tenant := as3Tenant{}
	ac := initDefaultAS3()
	adc := ac[DeclarationKey].(as3ADC)
	tenant.initDefault(partition)
	tenant[SharedKey] = shareApplication
	adc[partition] = tenant
	//remove Common if partition is not Common
	if IsSupportRouteDomain() && partition != DefaultPartition {
		delete(adc, DefaultPartition)
	}
	ac[DeclarationKey] = adc
	return ac
}

func (ac *as3Post) generateAS3ResourceDeclaration(adc as3ADC) {
	tenant := ac.tenantConfig.Name
	sharedApp := adc.getAS3SharedApp(tenant)
	if sharedApp == nil {
		adc.initDefault(tenant)
		sharedApp = adc.getAS3SharedApp(tenant)
	}
	ac.processResourcesForAS3(sharedApp)
	return
}

func (ac *as3Post) processResourcesForAS3(sharedApp as3Application) {

	//Create policies
	ac.newPoliciesDecl(sharedApp)

	//Create gw pools
	ac.newGWPoolDecl(sharedApp)

	//Create log pools
	ac.newLogPoolDecl(sharedApp)

	//Create VS ARP
	ac.newVirtualAddressDecl(sharedApp)

	//Create AS3 Service for virtual server
	ac.newServiceDecl(sharedApp)

	// create nat policy
	ac.newNatPoliciesDecl(sharedApp)
}

func (ac *as3Post) getEndpointMap() map[string]corev1.Endpoints {
	ret := make(map[string]corev1.Endpoints)
	for _, ep := range ac.endpointList.Items {
		ret[ep.Namespace+"/"+ep.Name] = ep
	}
	return ret
}

func (ac *as3Post) newNatPoliciesDecl(sharedApp as3Application) {
	epMap := ac.getEndpointMap()

	var natRules []NatRule
	// 实际ac.externalIPRuleList.Items就一个数据
	for _, eipRule := range ac.externalIPRuleList.Items {
		// dest addr
		var destAddrAttr string
		if len(eipRule.Spec.DestinationMatch.Addresses) > 0 {
			destAddrAttr = getAs3NatRuleListAttr(eipRule.Namespace, eipRule.Name, "dest_addr"+"_"+eipRule.Spec.DestinationMatch.Name)
			newFirewallAddressList(destAddrAttr, eipRule.Spec.DestinationMatch.Addresses, sharedApp)
		}

		// dest port
		var destPortAttr string
		if len(eipRule.Spec.DestinationMatch.Ports.Ports) > 0 {
			destPortAttr = getAs3NatRuleListAttr(eipRule.Namespace, eipRule.Name, "dest_port"+"_"+eipRule.Spec.DestinationMatch.Name)
			newFirewallPortsList(destPortAttr, eipRule.Spec.DestinationMatch.Ports.Ports, sharedApp)
		}

		// src translation
		srcTransAttr := getAs3NatRuleListAttr(eipRule.Namespace, eipRule.Name, "src_trans")
		newNatSourceTranslation(srcTransAttr, eipRule.Spec.ExternalAddresses, sharedApp)

		protocol := eipRule.Spec.DestinationMatch.Ports.Protocol
		if protocol == "" {
			protocol = "any"
		}
		natRule := NatRule{
			Name:     getAs3NatRuleListAttr(eipRule.Namespace, eipRule.Name, ""),
			Protocol: protocol,
			Source: &Source{
				AddressLists: make([]Use, 0, len(eipRule.Spec.Services)),
			},
			SourceTranslation: Use{Use: getAs3UsePathForPartition(ac.tenantConfig.Name, srcTransAttr)},
		}
		if destAddrAttr != "" {
			natRule.Destination = &Destination{
				AddressLists: []Use{
					{Use: getAs3UsePathForPartition(ac.tenantConfig.Name, destAddrAttr)},
				},
			}
		}
		if destPortAttr != "" {
			if natRule.Destination != nil {
				natRule.Destination.PortLists = []Use{
					{Use: getAs3UsePathForPartition(ac.tenantConfig.Name, destPortAttr)},
				}
			} else {
				natRule.Destination = &Destination{
					PortLists: []Use{
						{Use: getAs3UsePathForPartition(ac.tenantConfig.Name, destPortAttr)},
					},
				}
			}
		}

		// src addr
		for _, svcName := range eipRule.Spec.Services {
			ep, ok := epMap[eipRule.Namespace+"/"+svcName]
			if !ok {
				continue
			}

			var srcIPs []string
			for _, subnet := range ep.Subsets {
				for _, addr := range subnet.Addresses {
					srcIPs = append(srcIPs, addr.IP)
				}
			}

			srcAddrAttr := getAs3SrcAddressAttr("snat", eipRule.Namespace, eipRule.Name, svcName)
			newFirewallAddressList(srcAddrAttr, srcIPs, sharedApp)

			natRule.Source.AddressLists = append(natRule.Source.AddressLists, Use{
				Use: getAs3UsePathForPartition(ac.tenantConfig.Name, srcAddrAttr),
			})
		}

		natRules = append(natRules, natRule)
	}

	// 新增automap规则
	// todo: AS3 不支持 sourceTranslation: "automap"
	newNatSourceTranslation(defaultSnatTranslation, getExternalIPAddresses(), sharedApp)
	natRules = append(natRules, NatRule{
		Name:              defaultSnatRule,
		Protocol:          "any",
		SourceTranslation: Use{Use: getAs3UsePathForPartition(ac.tenantConfig.Name, defaultSnatTranslation)},
	})
	sharedApp[defaultSnatPolicy] = newNatPolicy(natRules)
}

func (ac *as3Post) newPoliciesDecl(sharedApp as3Application) {
	//create fw rule list map
	policyMap := ac.newRulesDecl(sharedApp)
	for tyns, ruleList := range policyMap {
		ty, ns := strings.Split(tyns, "|")[0], strings.Split(tyns, "|")[1]
		tntcfg := &TenantConfig{}
		if ns == "" {
			tntcfg = GetTenantConfigForParttition(DefaultPartition)
		} else {
			tntcfg = GetTenantConfigForNamespace(ns)
		}
		if tntcfg == nil {
			continue
		}
		as3PolicyAttr := getAs3PolicyAttr(ty, tntcfg.RouteDomain.Name)
		policy := newFirewallPolicy()
		//cache old policy
		if sharedApp[as3PolicyAttr] != nil {
			policy = sharedApp[as3PolicyAttr].(FirewallPolicy)
		}
		//policy.Rules = append(policy.Rules, ruleList...)
		policy.Rules = append(ruleList, policy.Rules...)
		sharedApp[as3PolicyAttr] = policy
	}
}

func (ac *as3Post) newRulesDecl(sharedApp as3Application) map[string][]Use {
	rules := ac.dealRule()
	/**
	{
		"ns|test1": [{}]
	}
	*/
	policyMap := map[string][]Use{}
	for _, rule := range rules {
		for _, evc := range rule.exsvcs {
			fwrList := newFirewallRuleList()
			as3SrcAddrAttr := ""
			if rule.ty == "ns" || rule.ty == "svc" {
				//exsvc update, need not focus on
				if len(rule.srcAddr) != 0 {
					//app add source address
					as3SrcAddrAttr = getAs3SrcAddressAttr(rule.ty, rule.namespace, rule.name, rule.epName)
					newFirewallAddressList(as3SrcAddrAttr, rule.srcAddr, sharedApp)
				}
			}
			//app add dest address
			as3DesAddrAttr := getAs3DestAddrAttr(rule.ty, rule.namespace, rule.name, evc.name)
			newFirewallAddressList(as3DesAddrAttr, evc.destAddress, sharedApp)
			//app add dest port
			for key, ports := range evc.destPorts {
				as3DestPortAddr := getAs3DestPortAttr(rule.ty, rule.namespace, rule.name, evc.name, key)
				//app add port
				if ports.protocol != "" {
					newFirewallPortsList(as3DestPortAddr, ports.ports, sharedApp)
				}
				//rule list add rule
				fwrList.Rules = append(fwrList.Rules, newFirewallRule(key, ports.protocol, rule.namespace, rule.action, evc.name, ports.irule,
					as3DesAddrAttr, as3DestPortAddr, as3SrcAddrAttr, rule.logging))
			}
			//app add rule list
			ruleListAttr := getAs3RuleListAttr(rule.ty, rule.namespace, rule.name, evc.name)
			sharedApp[ruleListAttr] = fwrList
			//app add policy
			tyns := fmt.Sprintf("%s|%s", rule.ty, rule.namespace)
			uses := policyMap[tyns]
			if uses == nil {
				uses = []Use{}
			}
			uses = append(uses, Use{getAs3UsePathForNamespace(rule.namespace, ruleListAttr)})
			policyMap[tyns] = uses
		}
	}
	return policyMap
}

func newFirewallPolicy() FirewallPolicy {
	return FirewallPolicy{
		Class: ClassFirewallPolicy,
		Rules: []Use{},
	}
}

func newFirewallRuleList() FirewallRuleList {
	return FirewallRuleList{
		Class: ClassFirewallRuleList,
		Rules: []FirewallRule{},
	}
}

func newNatPolicy(rules []NatRule) NatPolicy {
	return NatPolicy{
		Class: ClassNatPolicy,
		Rules: rules,
	}
}

func newFirewallRule(fwrName, protocol, namespace, action, exsvcName, irule, destAddrAttr, destPortAttr, srcAddrAttr string, logging bool) FirewallRule {
	rule := FirewallRule{
		Protocol: protocol,
		Action:   action,
		Name:     fmt.Sprintf("%s_%s_%s", action, exsvcName, fwrName),
		Destination: FirewallDestination{
			AddressLists: []Use{
				Use{
					getAs3UsePathForNamespace(namespace, destAddrAttr),
				},
			},
			PortLists: []Use{
				Use{
					getAs3UsePathForNamespace(namespace, destPortAttr),
				},
			},
		},
		LoggingEnabled: logging,
	}
	if protocol == "" {
		rule.Destination.PortLists = nil
	}
	if irule != "" {
		rule.IRule = &IRule{
			Bigip: fmt.Sprintf("/Common/%s", irule),
		}
	}
	if srcAddrAttr != "" {
		rule.Source = FirewallSource(FirewallDestination{
			AddressLists: []Use{
				Use{
					getAs3UsePathForNamespace(namespace, srcAddrAttr),
				},
			},
		})
	}
	return rule
}

func newFirewallAddressList(attr string, addresses []string, shareApp as3Application) {
	// have domain, set fqdns
	ips, dns := []string{}, []string{}
	for _, addr := range addresses {
		//ns src addr is cidr
		if _, _, err := net.ParseCIDR(addr); err == nil {
			ips = append(ips, addr)
			continue
		}
		ip := net.ParseIP(addr)
		if ip == nil {
			dns = append(dns, addr)
		} else {
			ips = append(ips, addr)
		}
	}
	// as3不允许addresses为nil或addresses的lenth为0。这里强制设置为1.1.1.1。
	if len(ips) == 0 {
		ips = []string{"1.1.1.1"}
	}
	shareApp[attr] = FirewallAddressList{
		Class:     ClassFirewallAddressList,
		Addresses: ips,
		Fqdns:     dns,
	}
}

func newFirewallPortsList(attr string, ports []string, shareApp as3Application) {
	shareApp[attr] = FirewallPortList{
		Class: ClassFirewallPortList,
		Ports: ports,
	}
}

// Create AS3 Pools
func (ac *as3Post) newGWPoolDecl(sharedApp as3Application) {
	serverAddresses := ac.tenantConfig.Gwpool.ServerAddresses
	pool := &Pool{
		Class: ClassPoll,
		Members: []Member{
			Member{
				ServerAddresses: serverAddresses,
				ServicePort:     0,
				Enable:          true,
			},
		},
		Monitors: []Monitor{
			Monitor{Bigip: "/Common/gateway_icmp"},
		},
	}
	sharedApp[getAs3GwPoolAttr()] = pool
}

func (ac *as3Post) newLogPoolDecl(sharedApp as3Application) {
	log := getLogPool()
	//Whether to configure logging profile
	if !isConfigLogProfile() {
		return
	}
	template := strings.ReplaceAll(log.Template, "k8s", getMasterCluster())
	template = strings.ReplaceAll(template, "{{tenant}}", ac.tenantConfig.Name)
	var logpool map[string]interface{}
	err := validateJSONAndFetchObject(template, &logpool)
	if err != nil {
		return
	}
	if !log.EnableRemoteLog {
		for k, v := range logpool {
			logpublish, ok := v.(map[string]interface{})
			if !ok {
				return
			}
			if logpublish[ClassKey] == ClassSecurityLogProfile {
				sharedApp[k] = logpublish
				continue
			}
			if logpublish[ClassKey] == ClassLogPublisher {
				logpublish["destinations"] = []map[string]interface{}{
					{
						"bigip": "/Common/local-db",
					},
				}
				sharedApp[k] = logpublish
			}
		}
	} else {
		for k, v := range logpool {
			sharedApp[k] = v
		}
	}
	//servicePort default is 514
	numbers := []Member{}
	if len(log.ServerAddresses) != 0 {
		for _, v := range log.ServerAddresses {
			ips := strings.Split(v, ":")
			ip := ips[0]
			port := 514
			if len(ips) > 1 {
				vs, err := strconv.Atoi(ips[1])
				if err == nil {
					port = vs
				}
			}
			numbers = append(numbers, Member{
				ServerAddresses: []string{ip},
				ServicePort:     port,
				Enable:          true,
			})
		}
	} else {
		numbers = append(numbers, Member{
			ServerAddresses: []string{"0.0.0.0"},
			ServicePort:     514,
			Enable:          true,
		})
	}
	sharedApp[getMasterCluster()+"_log_pool"] = &Pool{
		Class:   ClassPoll,
		Members: numbers,
		Monitors: []Monitor{
			Monitor{Bigip: fmt.Sprintf("/%s/%s", DefaultPartition, log.HealthMonitor)},
		},
	}
}

// Create VS ARP
func (ac *as3Post) newVirtualAddressDecl(sharedApp as3Application) {
	virtualAddress := ac.tenantConfig.VirtualService.VirtualAddresses.VirtualAddress
	if len(virtualAddress) == 0 {
		virtualAddress = "0.0.0.0"
	}
	//Enhance the ARP control ability of VS's virtualaddress
	//virtualAddress of VA use first value if config one address in VirtualAddresses of VS
	defaultVa := &VirtualServerVa{
		Class:          ClassServiceAddress,
		VirtualAddress: virtualAddress,
		IcmpEcho:       "disable",
		ArpEnabled:     false,
	}
	vaTemplate := ac.tenantConfig.VirtualService.VirtualAddresses.template
	if strings.TrimSpace(vaTemplate) != "" {
		va := map[string]interface{}{}
		err := validateJSONAndFetchObject(vaTemplate, &va)
		if err == nil {
			sharedApp[getAs3VsVaAttr()] = defaultVa
		}
	}
	if _, ok := sharedApp[getAs3VsVaAttr()]; !ok {
		virtualAddresses := ac.tenantConfig.VirtualService.VirtualAddresses
		if virtualAddresses.VirtualAddress != "" {
			defaultVa.VirtualAddress = virtualAddresses.VirtualAddress
		}
		if virtualAddresses.IcmpEcho != "" {
			defaultVa.IcmpEcho = virtualAddresses.IcmpEcho
		}
		defaultVa.ArpEnabled = virtualAddresses.ArpEnabled
	}
	sharedApp[getAs3VsVaAttr()] = defaultVa
}

// Create AS3 Service for Route
func (ac *as3Post) newServiceDecl(sharedApp as3Application) {
	svcPolicyPath := getAs3UsePathForPartition(ac.tenantConfig.Name, getAs3PolicyAttr("svc", ac.tenantConfig.RouteDomain.Name))
	enableSecurityLog := false
	//Whether to configure logging profile
	if isConfigLogProfile() {
		enableSecurityLog = true
	}
	if strings.TrimSpace(ac.tenantConfig.VirtualService.Template) != "" {
		//The property names of vs are mainly managed clusters
		vsTemplate := strings.ReplaceAll(ac.tenantConfig.VirtualService.Template, "k8s", getMasterCluster())
		vsTemplate = strings.ReplaceAll(vsTemplate, "{{tenant}}", ac.tenantConfig.Name)

		vs := map[string]interface{}{}
		err := validateJSONAndFetchObject(vsTemplate, &vs)
		if err == nil {
			vs[PolicyFirewallEnforcedKey] = Use{
				svcPolicyPath,
			}
			vs["pool"] = getAs3GwPoolAttr()
			if !enableSecurityLog {
				delete(vs, "securityLogProfiles")
			}
			vs["virtualAddresses"] = []Use{
				{
					getAs3UsePathForPartition(ac.tenantConfig.Name, getAs3VsVaAttr()),
				},
			}
			sharedApp[getAs3VSAttr()] = vs
			return
		}
	}
	//error not nil or template is '', set default
	defaultVs := &VirtualServer{
		Layer4:                 "any",
		TranslateServerAddress: false,
		TranslateServerPort:    false,
		VirtualAddresses: []Use{
			{
				getAs3UsePathForPartition(ac.tenantConfig.Name, getAs3VsVaAttr()),
			},
		},
		PolicyFirewallEnforced: Use{
			svcPolicyPath,
		},
		SecurityLogProfiles: []Use{{getAs3UsePathForPartition(ac.tenantConfig.Name, strings.ReplaceAll("k8s_afm_hsl_log_profile", "k8s", getMasterCluster()))}},
		VirtualPort:         0,
		Snat:                "none", // todo: 测试
		Class:               ClassVirtualServerL4,
		PolicyNAT: Use{
			Use: getAs3UsePathForPartition(ac.tenantConfig.Name, "k8s_snat_policy"),
		},
	}
	if !enableSecurityLog {
		defaultVs.SecurityLogProfiles = []Use{}
	}
	sharedApp[getAs3VSAttr()] = defaultVs
}

func (adc as3ADC) initDefault(partition string) {
	tenant := as3Tenant{}
	tenant.initDefault(partition)
	adc[partition] = tenant
}

func (adc as3ADC) getAS3Partition(partition string) as3Tenant {
	if adc[partition] == nil {
		return nil
	}
	tnt := adc[partition]
	switch tnt.(type) {
	case as3Tenant, as3JSONWithArbKeys:
		return tnt.(as3Tenant)
	case map[string]interface{}:
		return as3Tenant(tnt.(map[string]interface{}))
	}
	return nil
}

func (adc as3ADC) getAS3SharedApp(partition string) as3Application {
	if tnt := adc.getAS3Partition(partition); tnt != nil {
		if app := tnt.getAS3SharedApp(); app != nil {
			return app
		}
	}
	return nil
}

func (t as3Tenant) initDefault(partition string) {
	tntcfg := GetTenantConfigForParttition(partition)
	app := as3Application{}
	app.initDefault(partition)
	t[ClassKey] = ClassTenant
	if IsSupportRouteDomain() && partition != DefaultPartition {
		t[DefaultRouteDomainKey] = tntcfg.RouteDomain.Id
	}
	t[SharedKey] = app
}

func (t as3Tenant) getAS3SharedApp() as3Application {
	if t[SharedKey] != nil {
		app := t[SharedKey]
		switch app.(type) {
		case as3Application, as3JSONWithArbKeys:
			return app.(as3Application)
		case map[string]interface{}:
			return as3Application(app.(map[string]interface{}))
		}
	}
	return nil
}

func (a as3Application) initDefault(partition string) {
	a[ClassKey] = ClassApplication
	a[TemplateKey] = SharedValue
	if partition == CommonKey {
		globalPolicyAttr := getAs3PolicyAttr("global", "")
		a[globalPolicyAttr] = newFirewallPolicy()
	}

	tntCfg := GetTenantConfigForParttition(partition)
	nsPolicyAttr := getAs3PolicyAttr("ns", tntCfg.RouteDomain.Name)
	a[nsPolicyAttr] = newFirewallPolicy()
	svcPolicyAttr := getAs3PolicyAttr("svc", tntCfg.RouteDomain.Name)
	a[svcPolicyAttr] = newFirewallPolicy()
	a.allDenyRuleList(partition, svcPolicyAttr)
}

func (a as3Application) allDenyRuleList(partition, attr string) {
	svcPolicy := a[attr]
	p, ok := svcPolicy.(FirewallPolicy)
	if ok {
		p.Rules = append(p.Rules, Use{getAs3UsePathForPartition(partition, getAllDenyRuleListAttr())})
	}
	a[attr] = p

	a[getAllDenyRuleListAttr()] = FirewallRuleList{
		Class: ClassFirewallRuleList,
		Rules: []FirewallRule{
			{
				Protocol:       "any",
				Name:           DenyAllRuleName,
				Destination:    FirewallDestination{},
				Source:         FirewallSource{},
				Action:         "drop",
				LoggingEnabled: false,
			},
		},
	}

}

func (ac *as3Post) dealRule() []ruleData {
	rules := []ruleData{}
	tntCfg := ac.tenantConfig

	if tntCfg.Name == DefaultPartition {
		//clusteregress
		for _, clsRule := range ac.clusterEgressList.Items {
			rule := ruleData{
				ty:      "global",
				name:    clsRule.Name,
				action:  clsRule.Spec.Action,
				logging: clsRule.Spec.Logging,
			}
			for _, clsExSvcName := range clsRule.Spec.ExternalServices {
				for _, exsvc := range ac.externalServiceList.Items {
					if clsExSvcName == exsvc.Name && exsvc.Namespace == GetClusterSvcExtNamespace() {
						rule.exsvcs = append(rule.exsvcs, dealExsvc(exsvc))
					}
				}
			}
			rules = append(rules, rule)
		}
	}

	//namespaceegress
	for _, nsRule := range ac.namespaceEgressList.Items {
		rule := ruleData{
			ty:        "ns",
			name:      nsRule.Name,
			namespace: nsRule.Namespace,
			action:    nsRule.Spec.Action,
			logging:   nsRule.Spec.Logging,
		}
		for _, clsExSvcName := range nsRule.Spec.ExternalServices {
			for _, exsvc := range ac.externalServiceList.Items {
				if clsExSvcName == exsvc.Name && exsvc.Namespace == nsRule.Namespace {
					rule.exsvcs = append(rule.exsvcs, dealExsvc(exsvc))
				}
			}
		}
		for _, ns := range ac.namespaceList.Items {
			if ns.Name == nsRule.Namespace {
				rule.srcAddr = []string{ns.Annotations[NamespaceCidr]}
			}
		}
		rules = append(rules, rule)
	}
	//serviceegress
	for _, svcRule := range ac.serviceEgressList.Items {
		rule := ruleData{
			ty:        "svc",
			name:      svcRule.Name,
			namespace: svcRule.Namespace,
			action:    svcRule.Spec.Action,
			logging:   svcRule.Spec.Logging,
		}
		for _, clsExSvcName := range svcRule.Spec.ExternalServices {
			for _, exsvc := range ac.externalServiceList.Items {
				if clsExSvcName == exsvc.Name && exsvc.Namespace == svcRule.Namespace {
					rule.exsvcs = append(rule.exsvcs, dealExsvc(exsvc))
				}
			}
		}
		for _, ep := range ac.endpointList.Items {
			if ep.Namespace == svcRule.Namespace && ep.Name == svcRule.Spec.Service {
				rule.epName = ep.Name
				for _, subset := range ep.Subsets {
					for _, ips := range subset.Addresses {
						rule.srcAddr = append(rule.srcAddr, ips.IP)
					}
				}
			}
		}
		rules = append(rules, rule)
	}
	return rules
}

func dealExsvc(exsvc v1alpha1.ExternalService) *exsvcDate {
	sv := &exsvcDate{
		name:        exsvc.Name,
		destAddress: exsvc.Spec.Addresses,
	}
	ptlMap := make(map[string]portIrule)
	for _, pt := range exsvc.Spec.Ports {
		if pt.Port != "" {
			ptl := strings.ToLower(pt.Protocol)
			key := ptl
			//to diff bwt is processed separately
			if v, ok := ptlMap[key]; ok {
				if v.irule != pt.Bandwidth {
					key = pt.Name
				}
			} else {
				if pt.Bandwidth != "" {
					key = pt.Name
				}
			}
			//if the ports has bwt, set the suffix of the key to "_bwt"
			ports := append(ptlMap[key].ports, strings.Split(pt.Port, ",")...)
			ptlMap[key] = portIrule{
				protocol: ptl,
				irule:    pt.Bandwidth,
				ports:    ports,
			}
		}
	}
	//No port does not need to create a port list
	if len(ptlMap) == 0 {
		ptlMap["any"] = portIrule{}
	}
	sv.destPorts = ptlMap
	return sv
}

func (svc *exsvcDate) fwrDestPost(ty, namespace, ruleName, exsvcName string, shareApp as3Application) {
	for ptl, v := range svc.destPorts {
		destAddrKey := getAs3DestPortAttr(ty, namespace, ruleName, exsvcName, ptl)
		shareApp[destAddrKey] = v
	}
}

func getAs3DestPortAttr(ty, namespace, ruleName, exsvcName, protocol string) string {
	ty_ns := ty + "_" + namespace
	if ty == "global" {
		ty_ns = ty
	}
	return fmt.Sprintf("%s_%s_%s_ext_%s_ports_%s", GetCluster(), ty_ns, ruleName, exsvcName, protocol)
}

func getAs3DestAddrAttr(ty, namespace, ruleName, exsvcName string) string {
	ty_ns := ty + "_" + namespace
	if ty == "global" {
		ty_ns = ty
	}
	return fmt.Sprintf("%s_%s_%s_ext_%s_address", GetCluster(), ty_ns, ruleName, exsvcName)
}

func getAs3SrcAddressAttr(ty, namespace, ruleName, endpointName string) string {
	ty_ns := ty + "_" + namespace
	if ty == "global" {
		return ""
	}
	if ty == "ns" {
		return fmt.Sprintf("%s_%s_%s_src_address", GetCluster(), ty_ns, ruleName)
	}
	return fmt.Sprintf("%s_%s_%s_ep_%s_src_address", GetCluster(), ty_ns, ruleName, endpointName)
}

func getAs3RuleListAttr(ty, namespace, ruleName, exsvcName string) string {
	ty_ns := ty + "_" + namespace
	if ty == "global" {
		ty_ns = ty
	}
	return fmt.Sprintf("%s_%s_%s_ext_%s_rule_list", GetCluster(), ty_ns, ruleName, exsvcName)
}

func getAs3NatRuleListAttr(namespace, name, extra string) string {
	ret := fmt.Sprintf("%s_snat_%s_%s", GetCluster(), namespace, name)
	if extra != "" {
		ret += "_" + extra
	}
	return ret
}

func getAs3PolicyAttr(ty, routeDoamin string) string {
	if ty == "global" {
		return fmt.Sprintf("%s_system_global_policy", getMasterCluster())
	}

	if !IsSupportRouteDomain() {
		routeDoamin = "rd"
	}
	return fmt.Sprintf("%s_%s_policy_%s", getMasterCluster(), ty, routeDoamin)
}

func getAllDenyRuleListAttr() string {
	return fmt.Sprintf("%s_svc_deny_all_rule_list", getMasterCluster())
}

func getAs3GwPoolAttr() string {
	return fmt.Sprintf("%s_gw_pool", getMasterCluster())
}

func getAs3VSAttr() string {
	return fmt.Sprintf("%s_outbound_vs", getMasterCluster())
}

func getAs3VsVaAttr() string {
	return fmt.Sprintf("%s_outbound_va", getMasterCluster())
}

func getExternalIPAddresses() []string {
	return getValue(externalIPAddressesKey).([]string)
}

func getAs3UsePathForPartition(partition, attr string) string {
	if attr == "" {
		return ""
	}
	if partition == "" {
		partition = DefaultPartition
	}
	return fmt.Sprintf("/%s/Shared/%s", partition, attr)
}

func getAs3UsePathForNamespace(namespace, attr string) string {
	if attr == "" {
		return ""
	}
	partition := ""
	if namespace == "" {
		partition = DefaultPartition
	} else {
		tntcfg := GetTenantConfigForNamespace(namespace)
		if tntcfg == nil {
			panic(fmt.Sprintf("Get the current rd configuration of namespace[%s] as nil", namespace))
		}
		partition = tntcfg.Name
	}

	return fmt.Sprintf("/%s/Shared/%s", partition, attr)
}

func getOriginAttrOfUsePath(use string) string {
	k := strings.Split(use, "/")
	if len(k) < 4 {
		return ""
	}
	return k[3]
}

func getLatestUsePath(use string) string {
	k := strings.Split(use, "/")
	if len(k) == 0 {
		return ""
	}
	return k[len(k)-1]
}

func translateAs3Declaration(decl interface{}) as3Declaration {
	switch decl.(type) {
	case string:
		return as3Declaration(decl.(string))
	default:
		obj, err := json.Marshal(decl)
		if err != nil {
			return ""
		}
		return as3Declaration(obj)
	}
}

func patchResouce(partition string, isDelete bool, srcAdc, deltaAdc as3ADC) interface{} {
	src := srcAdc.getAS3SharedApp(partition)
	delta := deltaAdc.getAS3SharedApp(partition)
	var pathBody = PatchBody{}
	if src == nil {
		if isDelete {
			return nil
		}
		return append(pathBody, PatchItem{
			Op:    OpAdd,
			Path:  fmt.Sprintf("/%s", partition),
			Value: deltaAdc.getAS3Partition(partition),
		})
	}
	srcApp, deltaApp := map[string]interface{}{}, map[string]interface{}{}
	validateJSONAndFetchObject(src, &srcApp)
	validateJSONAndFetchObject(delta, &deltaApp)

	for deltaKey, deltaValue := range deltaApp {
		if srcValue, ok := srcApp[deltaKey]; ok {
			//filter out app attributes
			child, ok := srcValue.(map[string]interface{})
			if !ok {
				continue
			}
			//is deny exist, skip
			if deltaKey == getAllDenyRuleListAttr() {
				continue
			}
			//poll and vs don,t delete, only add or modify
			if isDelete {
				if skipDeleteShareApplicationClassOrAttr(partition, deltaKey) {
					continue
				}
				continue
			}
			if child[ClassKey].(string) == ClassFirewallPolicy {
				//to modify deltaValue, so pass param is pointer
				pathBody = policyPatchJson(srcValue, deltaValue, getAs3UsePathForPartition(partition, deltaKey), pathBody, isDelete)

			} else {
				patchItem := PatchItem{
					Op:    OpReplace,
					Path:  getAs3UsePathForPartition(partition, deltaKey),
					Value: deltaValue,
				}
				if isDelete {
					patchItem.Op = OpRemove
					patchItem.Value = nil
					pathBody = append(pathBody, patchItem)
				} else {
					if !reflect.DeepEqual(deltaValue, srcValue) {
						pathBody = append(pathBody, patchItem)
					}
				}
			}
		} else {
			if !isDelete {
				patchItem := PatchItem{
					Op:    OpAdd,
					Path:  getAs3UsePathForPartition(partition, deltaKey),
					Value: deltaValue,
				}
				pathBody = append(pathBody, patchItem)
			}
		}
	}
	return pathBody
}

func policyPatchJson(src, delta interface{}, path string, patchBody PatchBody, isDelete bool) PatchBody {
	srcData, err := json.Marshal(src)
	if err != nil {
		return patchBody
	}
	deltaData, err := json.Marshal(delta)
	if err != nil {
		return patchBody
	}
	srcPolicy, deltaPolicy := FirewallPolicy{}, FirewallPolicy{}
	err = json.Unmarshal(srcData, &srcPolicy)
	if err != nil {
		return patchBody
	}
	err = json.Unmarshal(deltaData, &deltaPolicy)
	if err != nil {
		return patchBody
	}
	if len(deltaPolicy.Rules) == 0 {
		return patchBody
	}

	val := Use{}
	//exclude denyall policy
	for _, v := range deltaPolicy.Rules {
		if strings.Contains(v.Use, getAllDenyRuleListAttr()) {
			continue
		}
		val = v
		break
	}
	//modify delta value

	for i, sr := range srcPolicy.Rules {
		if val.Use == sr.Use {
			if isDelete {
				//find ,remove this rule list in policy
				patchBody = append(patchBody, PatchItem{
					Op:   OpRemove,
					Path: fmt.Sprintf("%s/rules/%d", path, i),
				})
				return patchBody
			}
			//find, do not update
			return patchBody
		}
	}
	//do not find, do not delete
	if isDelete {
		return patchBody
	}
	return append(patchBody, PatchItem{
		Op:    OpAdd,
		Path:  fmt.Sprintf("%s/rules/-", path),
		Value: val,
	})
}

func fullResource(partition string, isDelete bool, srcAdc, deltaAdc as3ADC) interface{} {
	src := srcAdc.getAS3SharedApp(partition)
	delta := deltaAdc.getAS3SharedApp(partition)
	if src == nil && !isDelete {
		return newAs3Obj(partition, delta)
	}
	//originApp: save old as3
	originApp, srcApp, deltaApp := map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}
	if err := validateJSONAndFetchObject(src, &originApp); err != nil {
		return nil
	}
	if err := validateJSONAndFetchObject(src, &srcApp); err != nil {
		return nil
	}
	if err := validateJSONAndFetchObject(delta, &deltaApp); err != nil {
		return srcAdc
	}
	for deltaKey, deltaValue := range deltaApp {
		if srcValue, ok := srcApp[deltaKey]; ok {
			switch deltaKey {
			case defaultSnatPolicy:
				// 不用考虑删除
				srcApp[deltaKey] = natPolicyMergeFullJson(srcValue, deltaValue, isDelete)
				continue
			case defaultSnatTranslation:
				// 不用考虑删除
				srcApp[deltaKey] = srcValue
				continue
			}

			child, ok := srcValue.(map[string]interface{})
			if !ok {
				continue
			}
			//poll and vs don,t delete, only add or modify
			if isDelete {
				//filter out app attributes
				if skipDeleteShareApplicationClassOrAttr(partition, deltaKey) {
					continue
				}
			}
			switch child[ClassKey].(string) {
			case ClassFirewallPolicy:
				srcApp[deltaKey] = policyMergeFullJson(srcValue, deltaValue, isDelete)
			default:
				if isDelete {
					delete(srcApp, deltaKey)
				} else {
					srcApp[deltaKey] = deltaValue
				}
			}
		} else {
			if !isDelete {
				srcApp[deltaKey] = deltaValue
			}
		}
	}
	clearUpUnreferencePolicy(srcApp)
	if !isDiff(originApp, srcApp) && !isDelete {
		return nil
	}
	return newAs3Obj(partition, srcApp)
}

func natPolicyMergeFullJson(src, delta interface{}, isDelete bool) interface{} {
	srcData, err := json.Marshal(src)
	if err != nil {
		return src
	}
	deltaData, err := json.Marshal(delta)
	if err != nil {
		return src
	}
	srcPolicy, deltaPolicy := NatPolicy{}, NatPolicy{}
	err = json.Unmarshal(srcData, &srcPolicy)
	if err != nil {
		return src
	}
	err = json.Unmarshal(deltaData, &deltaPolicy)
	if err != nil {
		return src
	}

	for _, deltaRule := range deltaPolicy.Rules {
		isExist := false
		for i := len(srcPolicy.Rules) - 1; i >= 0; i-- {
			srcRule := srcPolicy.Rules[i]
			if deltaRule.Name == srcRule.Name {
				isExist = true

				// 跳过automap
				if srcRule.Name == defaultSnatRule {
					srcPolicy.Rules[i] = deltaRule
					continue
				}

				if isDelete {
					// 删除rule
					srcPolicy.Rules = append(srcPolicy.Rules[:i], srcPolicy.Rules[i+1:]...)
				} else {
					srcPolicy.Rules[i] = deltaRule
				}
				break
			}
		}
		if !isExist && !isDelete {
			srcPolicy.Rules = append(srcPolicy.Rules, deltaRule)
		}
	}

	// todo: rule排序

	//automap needs to be at the end
	last := len(srcPolicy.Rules) - 1
	for i := last; i >= 0; i-- {
		if strings.Contains(srcPolicy.Rules[i].Name, defaultSnatRule) {
			srcPolicy.Rules[i], srcPolicy.Rules[last] = srcPolicy.Rules[last], srcPolicy.Rules[i]
			break
		}
	}

	return srcPolicy
}

func policyMergeFullJson(src, delta interface{}, isDelete bool) interface{} {
	srcData, err := json.Marshal(src)
	if err != nil {
		return src
	}
	deltaData, err := json.Marshal(delta)
	if err != nil {
		return src
	}
	srcPolicy, deltaPolicy := FirewallPolicy{}, FirewallPolicy{}
	err = json.Unmarshal(srcData, &srcPolicy)
	if err != nil {
		return src
	}
	err = json.Unmarshal(deltaData, &deltaPolicy)
	if err != nil {
		return src
	}
	for _, deltaRule := range deltaPolicy.Rules {
		isExist := false
		for i, srcRule := range srcPolicy.Rules {
			if deltaRule.Use == srcRule.Use {
				isExist = true
				//if find, delete
				if isDelete {
					//skip  deny all in svc policy
					if strings.Contains(deltaRule.Use, getAllDenyRuleListAttr()) {
						continue
					}
					srcPolicy.Rules = append(srcPolicy.Rules[:i], srcPolicy.Rules[i+1:]...)
				}
				break
			}
		}
		if !isExist && !isDelete {
			srcPolicy.Rules = append(srcPolicy.Rules, deltaRule)
		}
	}
	//deny all needs to be at the end
	last := len(srcPolicy.Rules) - 1
	for i := last; i >= 0; i-- {
		if strings.Contains(srcPolicy.Rules[i].Use, getAllDenyRuleListAttr()) {
			denyAllRule := srcPolicy.Rules[i].Use
			srcPolicy.Rules[i].Use = srcPolicy.Rules[last].Use
			srcPolicy.Rules[last].Use = denyAllRule
			break
		}
	}
	return srcPolicy
}

func clearUpUnreferencePolicy(shareApp map[string]interface{}) {
	flag1 := map[string]bool{}
	flag2 := map[string]bool{}
	for key, value := range shareApp {
		// snat policy相关的数据结构与原有ces不同
		switch key {
		case defaultSnatPolicy:
			natPolicy, ok := value.(NatPolicy)
			if !ok {
				continue
			}

			for _, rule := range natPolicy.Rules {
				if rule.Destination != nil {
					// dest addr
					for _, destAddr := range rule.Destination.AddressLists {
						flag1[getOriginAttrOfUsePath(destAddr.Use)] = true
					}

					// dest port
					for _, destPort := range rule.Destination.PortLists {
						flag1[getOriginAttrOfUsePath(destPort.Use)] = true
					}
				}

				if rule.Source != nil {
					// src addr
					for _, srcAddr := range rule.Source.AddressLists {
						flag1[getOriginAttrOfUsePath(srcAddr.Use)] = true
					}
				}

				// src translation
				// automap 命名规则有区别，特殊处理
				flag1[getLatestUsePath(rule.SourceTranslation.Use)] = true
			}
			continue
		}

		obj, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		class := obj[ClassKey]
		if class == nil {
			continue
		}
		switch class.(string) {
		case ClassFirewallRuleList:
			rules := obj["rules"].([]interface{})
			for _, rule := range rules {
				dest := rule.(map[string]interface{})["destination"].(map[string]interface{})
				if addressList, ok := dest["addressLists"]; ok {
					for _, uses := range addressList.([]interface{}) {
						use := getOriginAttrOfUsePath(uses.(map[string]interface{})["use"].(string))
						flag1[use] = true
					}
				}
				if portList, ok := dest["portLists"]; ok {
					for _, uses := range portList.([]interface{}) {
						use := getOriginAttrOfUsePath(uses.(map[string]interface{})["use"].(string))
						flag1[use] = true
					}
				}
				//source address
				if src, ok := rule.(map[string]interface{})["source"].(map[string]interface{}); ok {
					if addressList, ok := src["addressLists"]; ok {
						for _, uses := range addressList.([]interface{}) {
							use := getOriginAttrOfUsePath(uses.(map[string]interface{})["use"].(string))
							flag1[use] = true
						}
					}
				}
			}
		case ClassFirewallAddressList, ClassFirewallPortList, ClassNatPolicy:
			flag2[key] = true
		case ClassSecurityLogProfile, ClassLogPublisher:
			if !isConfigLogProfile() {
				delete(shareApp, key)
			}
		}
	}

	for k, _ := range flag2 {
		if _, ok := flag1[k]; !ok {
			delete(shareApp, k)
		}
	}
}

func isDiff(old, new interface{}) bool {
	oldObj, newObj := map[string]interface{}{}, map[string]interface{}{}
	if err := validateJSONAndFetchObject(old, &oldObj); err != nil {
		return true
	}
	if err := validateJSONAndFetchObject(new, &newObj); err != nil {
		return true
	}
	for k, v := range newObj {
		v1, ok := oldObj[k]
		if !ok {
			return true
		}
		if !reflect.DeepEqual(v, v1) {
			return true
		}
	}
	return false
}

func validateJSONAndFetchObject(obj interface{}, jsonObj *map[string]interface{}) error {
	jsonData := ""
	switch obj.(type) {
	case string:
		jsonData = obj.(string)
		if jsonData == "" {
			klog.Errorf("obj json is empty string !!!")
			return fmt.Errorf("Empty Input JSON String")
		}
	case as3Declaration:
		jsonData = string(obj.(as3Declaration))
		if jsonData == "" {
			klog.Errorf("obj json is empty string !!!")
			return fmt.Errorf("Empty Input JSON String")
		}
	default:
		if obj == nil {
			klog.Errorf("obj json is nil !!!")
			return fmt.Errorf("Empty Input JSON String")
		}
		data, err := json.Marshal(obj)
		if err != nil {
			klog.Errorf(" obj json marshal error !!!")
			return fmt.Errorf("Empty Input JSON String")
		}
		jsonData = string(data)
	}
	if err := json.Unmarshal([]byte(jsonData), jsonObj); err != nil {
		klog.Errorf("Failed in JSON Un-Marshal test !!!: %v", err)
		return err
	}

	if data, err := json.Marshal(*jsonObj); err != nil && string(data) != "" {
		klog.Errorf("Failed in JSON Marshal test  !!!: %v", err)
		return err
	}

	return nil
}
