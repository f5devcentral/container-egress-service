package controller

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type (
	destPortsMap   map[string]map[string][]string
	destAddressMap map[string][]string
)

type egress struct {
	name      string
	namespace string
	exsvcs    []v1alpha1.ExternalService
	k8sSvc    string
	action    string
	ruleType  string
	isDelete  bool
}

func (c *Controller) extsvcsDestData(externalServices []v1alpha1.ExternalService, ruleType string) (destPortsMap,
	destAddressMap, error) {

	// service - protocol - ports
	destPorts := make(map[string]map[string][]string)

	//service - addrs
	destAddress := make(map[string][]string)

	for _, exsvc := range externalServices {
		svc, err := c.externalServicesLister.ExternalServices(exsvc.Namespace).Get(exsvc.Name)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, nil, err
			}
			klog.Warningf("externalService[%s/%s] does not exist", exsvc.Namespace, exsvc.Name)
			continue
		}

		//update ext ruleType namespace
		if svc.Labels == nil {
			svc.Labels = make(map[string]string, 1)
		}

		if svc.Labels[as3.RuleTypeLabel] != ruleType {
			svc.Labels[as3.RuleTypeLabel] = ruleType
			_, err = c.as3clientset.KubeovnV1alpha1().ExternalServices(exsvc.Namespace).Update(context.Background(), svc, metav1.UpdateOptions{})
			if err != nil {
				return nil, nil, err
			}

		}

		portsMap := make(map[string][]string)
		for _, port := range svc.Spec.Ports {
			if port.Port != "" {
				protocol := strings.ToLower(port.Protocol)
				portsMap[protocol] = append(portsMap[protocol], strings.Split(port.Port, ",")...)
			}
		}

		if len(portsMap) == 0 {
			portsMap["any"] = nil
		}
		destPorts[exsvc.Name] = portsMap

		destAddress[exsvc.Name] = svc.Spec.Addresses
	}
	return destPorts, destAddress, nil

}

func (c *Controller) pkgEgress(eg egress, as3Namespace *as3.As3Namespace) (string, []as3.PatchItem, error) {
	as3RulesList, patchBody := make([]as3.PatchItem, 0), make([]as3.PatchItem, 0)
	policePath := ""

	cluster := as3.GetCluster()
	pathProfix := as3.AS3PathPrefix(as3Namespace)
	ruleName, namespace, action, ruleType, exsvcs := eg.name, eg.namespace, eg.action, eg.ruleType, eg.exsvcs
	destPorts, destAddress, err := c.extsvcsDestData(exsvcs, ruleType)
	if err != nil {
		return policePath, patchBody, err
	}

	ty_ns := ""
	switch ruleType {
	case as3.RuleTypeGlobal:
		ty_ns = "global"
		policePath = fmt.Sprintf("/Common/Shared/%s_system_global_policy", cluster)

	case as3.RuleTypeNamespace:
		policePath = fmt.Sprintf("%s_ns_policy_%s", pathProfix, as3Namespace.RouteDomain.Name)
		if !as3.IsSupportRouteDomain() {
			//because only one police
			policePath = fmt.Sprintf("/Common/Shared/%s_ns_policy_rd", cluster)
		}

		ty_ns = "ns_" + namespace
		//get ns
		ns, err := c.kubeclientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get namespace[%s],due to: %v", namespace, err)
			return policePath, patchBody, err
		}
		subnet := ns.Annotations["ovn.kubernetes.io/cidr"]
		srcAddrList := as3.FirewallAddressList{
			Class: as3.ClassFirewallAddressList,
		}
		srcAddrList.Addresses = append(srcAddrList.Addresses, subnet)
		patchItem := as3.PatchItem{
			Path:  fmt.Sprintf("%s_%s_%s_src_addr", pathProfix, ty_ns, ruleName),
			Op:    as3.OpAdd,
			Value: srcAddrList,
		}
		patchBody = append(patchBody, patchItem)
	case as3.RuleTypeService:
		policePath = fmt.Sprintf("%s_svc_policy_%s", pathProfix, as3Namespace.RouteDomain.Name)
		if !as3.IsSupportRouteDomain() {
			//because only one police
			policePath = fmt.Sprintf("/Common/Shared/%s_svc_policy_rd", cluster)
		}

		ty_ns = "svc_" + namespace
		//get ep
		ep, err := c.endpointsLister.Endpoints(namespace).Get(eg.k8sSvc)
		if err != nil {
			klog.Errorf("failed to get endpoint [%s/%s],due to: %v", namespace, eg.k8sSvc, err)
			return policePath, patchBody, err
		}

		srcAddrList := as3.FirewallAddressList{
			Class: as3.ClassFirewallAddressList,
		}
		//get src ip
		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				srcAddrList.Addresses = append(srcAddrList.Addresses, addr.IP)
			}
		}
		patchItem := as3.PatchItem{
			Path:  fmt.Sprintf("%s_%s_%s_src_addr_%s", pathProfix, ty_ns, ruleName, eg.k8sSvc),
			Op:    as3.OpAdd,
			Value: srcAddrList,
		}
		patchBody = append(patchBody, patchItem)
	}

	for svcName, portMap := range destPorts {
		//FirewallAddressList
		addrs := destAddress[svcName]
		value := as3.FirewallAddressList{
			Class:     as3.ClassFirewallAddressList,
			Addresses: addrs,
		}
		addrItem := as3.PatchItem{
			Path:  fmt.Sprintf("%s_%s_%s_ext_%s_address", pathProfix, ty_ns, ruleName, svcName),
			Op:    as3.OpAdd,
			Value: value,
		}

		//FirewallRuleList
		as3Rules := as3.FirewallRuleList{
			Class: as3.ClassFirewallRuleList,
			Rules: []as3.FirewallRule{},
		}
		portList := make([]as3.PatchItem, 0)
		for protocol, ports := range portMap {
			//FirewallPortList
			value := as3.FirewallPortList{
				Class: as3.ClassFirewallPortList,
				Ports: ports,
			}
			portItem := as3.PatchItem{
				Path:  fmt.Sprintf("%s_%s_%s_ext_%s_ports_%s", pathProfix, ty_ns, ruleName, svcName, protocol),
				Op:    as3.OpAdd,
				Value: value,
			}
			portList = append(portList, portItem)
			//FirewallRule
			//global police don't have source, ns policy source addr: subnet, svc policy source addr: pod ips
			as3Rule := as3.FirewallRule{
				Name:     fmt.Sprintf("%s_%s_%s", action, svcName, protocol),
				Protocol: protocol,
				Action:   action,
				Destination: as3.FirewallDestination{
					PortLists: []as3.Use{
						{Use: portItem.Path},
					},
					AddressLists: []as3.Use{
						{Use: addrItem.Path},
					},
				},
			}
			if ruleType != as3.RuleTypeGlobal {
				path := fmt.Sprintf("%s_%s_%s_src_addr_%s", pathProfix, ty_ns, ruleName, eg.k8sSvc)
				if ruleType == as3.RuleTypeNamespace {
					path = fmt.Sprintf("%s_%s_%s_src_addr", pathProfix, ty_ns, ruleName)
				}
				as3Rule.Source = as3.FirewallSource{
					AddressLists: []as3.Use{
						{Use: path},
					},
				}
			}
			as3Rules.Rules = append(as3Rules.Rules, as3Rule)
		}
		as3RulesItem := as3.PatchItem{
			Path:  fmt.Sprintf("%s_%s_%s_ext_%s_rule_list", pathProfix, ty_ns, ruleName, svcName),
			Op:    as3.OpAdd,
			Value: as3Rules,
		}
		as3RulesList = append(as3RulesList, as3RulesItem)
		patchBody = append(patchBody, addrItem, as3RulesItem)
		patchBody = append(patchBody, portList...)
	}

	tenant := as3Namespace.Parttion
	// get AS3 declaration
	isExistTenant := true
	adc, err := c.as3Client.Get(tenant)
	if err != nil {
		//Common isn't exist, return error
		if ruleType == as3.RuleTypeGlobal {
			return policePath, patchBody, err
		}
		if as3.IsNotFound(err) {
			isExistTenant = false
		} else {
			return policePath, patchBody, fmt.Errorf("failed to get BIG-IP: %v", err)
		}
	}

	//add tenant
	if !isExistTenant {
		nsPolicy := as3.FirewallPolicy{
			Class: as3.ClassFirewallPolicy,
			Rules: []as3.Use{},
		}
		for _, as3Rule := range as3RulesList {
			nsPolicy.Rules = append(nsPolicy.Rules, as3.Use{Use: as3Rule.Path})
		}
		policyItem := as3.PatchItem{
			Path:  policePath,
			Op:    as3.OpAdd,
			Value: nsPolicy,
		}

		patchBody = append(patchBody, policyItem)

		if ruleType == as3.RuleTypeNamespace {
			//to add deny all policy
			svcPolicyItem := as3.PatchItem{
				Path: fmt.Sprintf("%s_svc_policy_%s", pathProfix, as3Namespace.RouteDomain.Name),
				Op:   as3.OpAdd,
				Value: as3.FirewallPolicy{
					Class: as3.ClassFirewallPolicy,
					Rules: []as3.Use{},
				},
			}
			patchBody = append(patchBody, svcPolicyItem)
		}

		as3Tenant, err := as3.NewAs3Tenant(as3Namespace, patchBody)
		if err != nil {
			return "", nil, err
		}

		as3Tenant[as3.DefaultRouteDomainKey] = as3Namespace.RouteDomain.Id
		tenantItem := as3.PatchItem{
			Op:    as3.OpAdd,
			Path:  "/" + tenant,
			Value: as3Tenant,
		}

		err = c.as3Client.Patch(tenantItem)
		if err != nil {
			err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
			klog.Error(err)
			return policePath, patchBody, err
		}
		klog.Infof("BIG-IP add %s tenant success", tenant)
	}

	//Determine to update the rules in all patch bodies
	patchBody = as3.JudgeSelectedUpdate(adc, patchBody, eg.isDelete)

	// find polices, if exists: policy have created

	jsonPath := strings.ReplaceAll(policePath, "/", ".")[1:]
	if ok := gjson.Get(adc, jsonPath).Exists(); ok {
		for _, as3Rule := range as3RulesList {
			policyRuleList := gjson.Get(adc, fmt.Sprintf("%s.rules", jsonPath)).Array()
			//find index the value of item.Path
			index := -1
			for i, rule := range policyRuleList {
				if rule.Get("use").String() == as3Rule.Path {
					index = i
					break
				}
			}
			policyItem := as3.PatchItem{
				Path: fmt.Sprintf("%s/rules/-", policePath),
				Value: as3.Use{
					Use: as3Rule.Path,
				},
			}
			//if isDelete is true( if exist: remove );
			if eg.isDelete {
				if index > -1 {
					policyItem.Op = as3.OpRemove
					policyItem.Path = fmt.Sprintf("%s/rules/%d", policePath, index)
					patchBody = append(patchBody, policyItem)
				}
			} else {
				//don,t exist: add
				if index == -1 {
					policyItem.Op = as3.OpAdd
					patchBody = append(patchBody, policyItem)
				}
			}
		}
	} else {
		policy := as3.FirewallPolicy{
			Class: as3.ClassFirewallPolicy,
			Rules: []as3.Use{},
		}
		for _, as3Rule := range as3RulesList {
			policy.Rules = append(policy.Rules, as3.Use{Use: as3Rule.Path})
		}
		policyItem := as3.PatchItem{
			Path:  policePath,
			Op:    as3.OpAdd,
			Value: policy,
		}
		patchBody = append(patchBody, policyItem)
	}

	if ruleType == as3.RuleTypeService {
		//svc policy add in vs
		vsPath := fmt.Sprintf("%s_outbound_vs", pathProfix)
		if !as3.IsSupportRouteDomain() {
			//because only one vs
			vsPath = fmt.Sprintf("/Common/Shared/%s_outbound_vs", cluster)
		}
		jsonVsPath := strings.ReplaceAll(vsPath, "/", ".")[1:]
		result := gjson.Get(adc, jsonVsPath)
		if !result.Exists() {
			vs, err := as3.NewVirtualServer(as3Namespace)
			if err != nil {
				klog.Errorf("NewVirtualServer failed: %v", err)
				return policePath, patchBody, err
			}

			patchVsItem := as3.PatchItem{
				Op:    as3.OpAdd,
				Path:  vsPath,
				Value: vs,
			}

			gwPoll := as3.NewPoll(as3Namespace.Gwpool.ServerAddresses)

			patchPollItem := as3.PatchItem{
				Op:    as3.OpAdd,
				Path:  fmt.Sprintf("/%s/Shared/%s", tenant, vs.Pool),
				Value: gwPoll,
			}
			patchBody = append(patchBody, patchPollItem, patchVsItem)
		} else {
			res := result.Map()[as3.PolicyFirewallEnforcedKey]
			if res.Exists() {
				if res.Map()["use"].String() != policePath {
					policeItem := as3.PatchItem{
						Op:   as3.OpReplace,
						Path: fmt.Sprintf("%s_outbound_vs/%s", pathProfix, as3.PolicyFirewallEnforcedKey),
						Value: as3.Use{
							Use: policePath,
						},
					}
					patchBody = append(patchBody, policeItem)
				}
			} else {
				policeItem := as3.PatchItem{
					Op:   as3.OpAdd,
					Path: fmt.Sprintf("%s_outbound_vs/%s", pathProfix, as3.PolicyFirewallEnforcedKey),
					Value: as3.Use{
						Use: policePath,
					},
				}
				patchBody = append(patchBody, policeItem)
			}
		}
	}

	return policePath, patchBody, nil
}

type syncFrequency struct {
	updateTimes []time.Time
	lock        sync.Mutex
}

var syncFq = syncFrequency{}

func (c *Controller) frequency() {
	syncFq.lock.Lock()
	defer syncFq.lock.Unlock()
	now := time.Now()
	isUpdateEpFq := func() bool {
		times := 0
		for _, v := range syncFq.updateTimes {
			if times > 5 {
				return true
			}
			if now.Sub(v) < 2*60*time.Second {
				times += 1
			}
		}
		return false
	}
	if len(syncFq.updateTimes) > 10 || isUpdateEpFq() {
		err := c.as3Client.StoreDisk()
		if err != nil {
			klog.Errorf("BIG-IP store disk error: %v", err)
			return
		}
		syncFq.updateTimes = []time.Time{}
	} else {
		syncFq.updateTimes = append(syncFq.updateTimes, now)
	}
}
