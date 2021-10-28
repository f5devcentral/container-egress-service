package as3

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

//other partition set clusterEgressList is nil
//The purpose is not to set the global policy
type as3Post struct {
	serviceEgressList   *v1alpha1.ServiceEgressRuleList
	namespaceEgressList *v1alpha1.NamespaceEgressRuleList
	clusterEgressList   *v1alpha1.ClusterEgressRuleList
	externalServiceList *v1alpha1.ExternalServiceList
	endpointList        *corev1.EndpointsList
	namespaceList       *corev1.NamespaceList
	tenantConfig        *TenantConfig
}

func newAs3Post(serviceEgressList *v1alpha1.ServiceEgressRuleList, namespaceEgressList *v1alpha1.NamespaceEgressRuleList,
	clusterEgressList *v1alpha1.ClusterEgressRuleList, externalServiceList *v1alpha1.ExternalServiceList,
	endpointList *corev1.EndpointsList, namespaceList *corev1.NamespaceList, tenantConfig *TenantConfig) *as3Post {
	//init default value, make sure not nil pointer
	ac := as3Post{
		serviceEgressList:   &v1alpha1.ServiceEgressRuleList{},
		namespaceEgressList: &v1alpha1.NamespaceEgressRuleList{},
		clusterEgressList:   &v1alpha1.ClusterEgressRuleList{},
		externalServiceList: &v1alpha1.ExternalServiceList{},
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
	tenant := as3Tenant{
		SharedKey: shareApplication,
		ClassKey:  TenantValue,
	}
	ac := initDefaultAS3()
	adc := ac[DeclarationKey].(as3ADC)
	adc[partition] = tenant
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

	//Create AS3 Service for virtual server
	ac.newServiceDecl(sharedApp)
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
		policy.Rules = append(policy.Rules, ruleList...)
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
			for ptl, ports := range evc.destPorts {
				as3DestPortAddr := getAs3DestPortAttr(rule.ty, rule.namespace, rule.name, evc.name, ptl)
				//app add port
				if ptl != "any"{
					newFirewallPortsList(as3DestPortAddr, ports.ports, sharedApp)
				}
				//rule list add rule
				fwrList.Rules = append(fwrList.Rules, newFirewallRule(ptl, rule.namespace, rule.action, evc.name, ports.irule,
					as3DesAddrAttr,
					as3DestPortAddr,
					as3SrcAddrAttr))
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

func newFirewallRule(protocol, namespace, action, exsvcName, irule, destAddrAttr, destPortAttr, srcAddrAttr string) FirewallRule {
	rule := FirewallRule{
		Protocol: protocol,
		Action:   action,
		Name:     fmt.Sprintf("%s_%s_%s", action, exsvcName, protocol),
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
		LoggingEnabled: true,
	}
	if protocol == "any"{
		rule.Destination.PortLists = nil
	}
	if irule != "" {
		rule.IRule = &IRule{
			Bigip: fmt.Sprintf("/Common/%s", irule),
		}
		rule.Protocol = strings.Replace(rule.Protocol, "_bwc", "", 1)
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
	if log.Template == "" {
		return
	}
	template := strings.ReplaceAll(log.Template, "k8s", GetCluster())
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
	sharedApp[GetCluster()+"_log_pool"] = &Pool{
		Class: ClassPoll,
		Members: []Member{
			Member{
				ServerAddresses: log.ServerAddresses,
				ServicePort:     0,
				Enable:          true,
			},
		},
		Monitors: []Monitor{
			Monitor{Bigip: "/Common/gateway_icmp"},
		},
	}
}

// Create AS3 Service for Route
func (ac *as3Post) newServiceDecl(sharedApp as3Application) {
	svcPolicyPath := getAs3UsePathForPartition(ac.tenantConfig.Name, getAs3PolicyAttr("svc", ac.tenantConfig.RouteDomain.Name))
	if ac.tenantConfig.VirtualService.Template != "" {
		vsTemplate := strings.ReplaceAll(ac.tenantConfig.VirtualService.Template, "k8s", GetCluster())
		vsTemplate = strings.ReplaceAll(vsTemplate, "{{tenant}}", ac.tenantConfig.Name)

		vs := map[string]interface{}{}
		err := validateJSONAndFetchObject(vsTemplate, &vs)
		if err != nil {
			//error not nil, set default
			sharedApp[getAs3VSAttr()] = &VirtualServer{
				Layer4:                 "any",
				TranslateServerAddress: false,
				TranslateServerPort:    false,
				VirtualAddresses:       []string{"0.0.0.0"},
				PolicyFirewallEnforced: Use{
					svcPolicyPath,
				},
				SecurityLogProfiles: []Use{{getAs3UsePathForPartition(ac.tenantConfig.Name, "k8s_afm_hsl_log_profile")}},
				VirtualPort:         0,
				Snat:                "auto",
				Class:               ClassVirtualServerL4,
				Pool:                getAs3GwPoolAttr(),
			}
			return
		}
		vs[PolicyFirewallEnforcedKey] = Use{
			svcPolicyPath,
		}
		vs["pool"] = getAs3GwPoolAttr()
		sharedApp[getAs3VSAttr()] = vs

	} else {
		VirtualAddresses := ac.tenantConfig.VirtualService.VirtualAddresses
		if len(ac.tenantConfig.VirtualService.VirtualAddresses) == 0 {
			VirtualAddresses = []string{"0.0.0.0"}
		}
		sharedApp[getAs3VSAttr()] = &VirtualServer{
			Layer4:                 "any",
			TranslateServerAddress: false,
			TranslateServerPort:    false,
			VirtualAddresses:       VirtualAddresses,
			PolicyFirewallEnforced: Use{
				svcPolicyPath,
			},
			SecurityLogProfiles: []Use{{getAs3UsePathForPartition(ac.tenantConfig.Name, "k8s_afm_hsl_log_profile")}},
			VirtualPort:         0,
			Snat:                "auto",
			Class:               ClassVirtualServerL4,
			Pool:                getAs3GwPoolAttr(),
		}
	}
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
	if partition != DefaultPartition {
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
				Protocol:    "any",
				Name:        DenyAllRuleName,
				Destination: FirewallDestination{},
				Source:      FirewallSource{},
				Action:      "drop",
				LoggingEnabled: true,
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
				ty:     "global",
				name:   clsRule.Name,
				action: clsRule.Spec.Action,
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
			if pt.Bandwidth != ""{
				key = fmt.Sprintf("%s_bwc", key)
			}
			//if the ports has bwt, set the suffix of the key to "_bwt"
			ports := append(ptlMap[key].ports, strings.Split(pt.Port, ",")...)
			ptlMap[key] = portIrule{
				irule: pt.Bandwidth,
				ports: ports,
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

func getAs3PolicyAttr(ty, routeDoamin string) string {
	if ty == "global" {
		return fmt.Sprintf("%s_system_global_policy", GetCluster())
	}

	if !IsSupportRouteDomain() {
		routeDoamin = "rd"
	}
	return fmt.Sprintf("%s_%s_policy_%s", GetCluster(), ty, routeDoamin)
}

func getAllDenyRuleListAttr() string {
	return fmt.Sprintf("%s_svc_deny_all_rule_list", GetCluster())
}

func getAs3GwPoolAttr() string {
	return fmt.Sprintf("%s_gw_pool", GetCluster())
}

func getAs3VSAttr() string {
	return fmt.Sprintf("%s_outbound_vs", GetCluster())
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
		partition = tntcfg.Name
	}

	return fmt.Sprintf("/%s/Shared/%s", partition, attr)
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
	srcApp, deltaApp := map[string]interface{}{}, map[string]interface{}{}
	if err := validateJSONAndFetchObject(src, &srcApp); err != nil {
		return nil
	}
	if err := validateJSONAndFetchObject(delta, &deltaApp); err != nil {
		return srcAdc
	}
	for deltaKey, deltaValue := range deltaApp {
		if srcValue, ok := srcApp[deltaKey]; ok {

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
			if child[ClassKey].(string) == ClassFirewallPolicy {
				srcApp[deltaKey] = policyMergeFullJson(srcValue, deltaValue, isDelete)
			} else {
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
	return newAs3Obj(partition, srcApp)
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
					if strings.Contains(deltaRule.Use, getAllDenyRuleListAttr()){
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
	return srcPolicy
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
