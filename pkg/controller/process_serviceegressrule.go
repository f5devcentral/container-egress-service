package controller

import (
	"context"
	"fmt"
	"strings"

	kubeovn "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func (c *Controller) processNextSeviceEgressRuleWorkerItem() bool {
	obj, shutdown := c.seviceEgressRuleWorkqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.seviceEgressRuleWorkqueue.Done(obj)

		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			c.seviceEgressRuleWorkqueue.Forget(obj)
			utilruntime.HandleError(err)
			return err
		}

		var rule *kubeovn.ServiceEgressRule
		var ok bool
		if rule, ok = obj.(*kubeovn.ServiceEgressRule); !ok {
			c.seviceEgressRuleWorkqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected seviceEgressRuleWorkqueue in workqueue but got %#v", obj))
			return nil
		}

		if err := c.serviceEgressRuleSyncHandler(key, rule); err != nil {
			c.seviceEgressRuleWorkqueue.AddRateLimited(rule)
			return fmt.Errorf("error syncing serviceEgressRule[%s]: %s, requeuing", key, err.Error())
		}

		c.seviceEgressRuleWorkqueue.Forget(obj)
		klog.Infof("Successfully synced serviceEgressRule[%s]", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) serviceEgressRuleSyncHandler(key string, rule *kubeovn.ServiceEgressRule) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	nsConfig := as3.GetConfigNamespace(namespace)
	if nsConfig == nil {
		klog.Infof("namespace[%s] not in watch range ", namespace)
		return nil
	}

	klog.Infof("===============================>start sync serviceEgressRule[%s/%s]", namespace, name)
	defer klog.Infof("===============================>end sync serviceEgressRule[%s/%s]", namespace, name)

	tenant := nsConfig.Parttion
	pathProfix := as3.AS3PathPrefix(nsConfig)
	//gw_pool.ServerAddresses
	serverAddresses := nsConfig.Gwpool.ServerAddresses
	routeDomain := nsConfig.RouteDomain

	var isDelete bool
	var r *kubeovn.ServiceEgressRule
	if r, err = c.seviceEgressRuleLister.ServiceEgressRules(namespace).Get(name); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		isDelete = true
	} else {
		rule = r
		if rule.Status.Phase != kubeovn.ServiceEgressRuleSyncing {
			rule.Status.Phase = kubeovn.ServiceEgressRuleSyncing
			rule, err = c.as3clientset.KubeovnV1alpha1().ServiceEgressRules(namespace).UpdateStatus(context.Background(), rule,
				v1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	defer func() {
		if err != nil {
			c.recorder.Event(rule, corev1.EventTypeWarning, err.Error(), MessageResourceFailedSynced)
		}
	}()

	// service - protocol - ports
	destPorts := make(map[string]map[string][]string)

	//service - addrs
	destAddress := make(map[string][]string)

	for _, svcName := range rule.Spec.ExternalServices {
		svc, err := c.externalServicesLister.ExternalServices(namespace).Get(svcName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			klog.Warningf("external service %s does not exist", svcName)
			continue
		}

		//update ext ruleType service
		if svc.Labels == nil {
			svc.Labels = make(map[string]string, 1)
		}

		if svc.Labels[as3.RuleTypeLabel] != as3.RuleTypeService {
			svc.Labels[as3.RuleTypeLabel] = as3.RuleTypeService
			_, err = c.as3clientset.KubeovnV1alpha1().ExternalServices(namespace).Update(context.Background(), svc, v1.UpdateOptions{})
			if err != nil {
				return err
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
		destPorts[svcName] = portsMap

		destAddress[svcName] = svc.Spec.Addresses
	}
	patchBody := make([]as3.PatchItem, 0)

	as3RulesList := make([]as3.PatchItem, 0)

	for svcName, portMap := range destPorts {
		//FirewallAddressList
		addrs := destAddress[svcName]
		value := as3.FirewallAddressList{
			Class:     as3.ClassFirewallAddressList,
			Addresses: addrs,
		}
		addrItem := as3.PatchItem{
			Path:  fmt.Sprintf("%s_svc_%s_%s_ext_%s_address", pathProfix, rule.Namespace, rule.Name, svcName),
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
				Path:  fmt.Sprintf("%s_svc_%s_%s_ext_%s_ports_%s", pathProfix, rule.Namespace, rule.Name, svcName, protocol),
				Op:    as3.OpAdd,
				Value: value,
			}
			portList = append(portList, portItem)
			//FirewallRule
			//extenal service police have source
			as3Rule := as3.FirewallRule{
				Name:     fmt.Sprintf("%s_%s_%s", rule.Spec.Action, svcName, protocol),
				Protocol: protocol,
				Action:   rule.Spec.Action,
				Destination: as3.FirewallDestination{
					PortLists: []as3.Use{
						{Use: portItem.Path},
					},
					AddressLists: []as3.Use{
						{Use: addrItem.Path},
					},
				},
				//service to source addr
				Source: as3.FirewallSource{
					AddressLists: []as3.Use{
						{Use: fmt.Sprintf("%s_svc_%s_%s_src_addr_%s", pathProfix, rule.Namespace, rule.Name, rule.Spec.Service)},
					},
				},
			}

			as3Rules.Rules = append(as3Rules.Rules, as3Rule)
		}
		as3RulesItem := as3.PatchItem{
			Path:  fmt.Sprintf("%s_svc_%s_%s_ext_%s_rule_list", pathProfix, rule.Namespace, rule.Name, svcName),
			Op:    as3.OpAdd,
			Value: as3Rules,
		}
		as3RulesList = append(as3RulesList, as3RulesItem)
		patchBody = append(patchBody, addrItem, as3RulesItem)
		patchBody = append(patchBody, portList...)
	}

	//get ep
	ep, err := c.endpointsLister.Endpoints(namespace).Get(rule.Spec.Service)
	if err != nil {
		klog.Errorf("failed to get endpoint [%s/%s],due to: %v", rule.Namespace, rule.Spec.Service, err)
		return err
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
		Path:  fmt.Sprintf("%s_svc_%s_%s_src_addr_%s", pathProfix, rule.Namespace, rule.Name, ep.Name),
		Op:    as3.OpAdd,
		Value: srcAddrList,
	}

	patchBody = append(patchBody, patchItem)

	svcRouteDomainPolicePath := fmt.Sprintf("%s_svc_policy_%s", pathProfix, routeDomain.Name)
	if !as3.GetAs3Config().IsSupportRouteDomain {
		//because only one svc police
		svcRouteDomainPolicePath = "/Common/Shared/k8s_svc_policy_rd"
	}
	// get AS3 declaration
	isExistTenant := true
	adc, err := c.as3Client.Get(tenant)
	if err != nil {
		if as3.IsNotFound(err) {
			isExistTenant = false
		} else {
			return fmt.Errorf("failed to get AS3: %v", err)
		}
	}

	//add tenant
	if !isExistTenant {
		policy := as3.FirewallPolicy{
			Class: as3.ClassFirewallPolicy,
			Rules: []as3.Use{},
		}
		for _, as3Rule := range as3RulesList {
			policy.Rules = append(policy.Rules, as3.Use{Use: as3Rule.Path})
		}
		policyItem := as3.PatchItem{
			Path:  svcRouteDomainPolicePath,
			Op:    as3.OpAdd,
			Value: policy,
		}
		patchBody = append(patchBody, policyItem)
		as3Tenant, err := as3.NewAs3Tenant(nsConfig, patchBody, true)
		if err != nil {
			return err
		}

		as3Tenant["defaultRouteDomain"] = routeDomain.Id
		tenantItem := as3.PatchItem{
			Op:    as3.OpAdd,
			Path:  "/" + tenant,
			Value: as3Tenant,
		}

		//search route domian
		url := fmt.Sprintf("/mgmt/tm/net/route-domain/~%s~%s", tenant, routeDomain.Name)
		_, err = c.as3Client.GetF5Resource(url)
		if err != nil {
			klog.Errorf("failed to get route domian %s, error:%v", routeDomain.Name, err)
			return err
		}

		err = c.as3Client.Patch(tenantItem)
		if err != nil {
			err = fmt.Errorf("failed to request AS3 Patch API: %v", err)
			klog.Error(err)
			return err
		}
		klog.Infof("as3 add %s tenant success", tenant)
		c.recorder.Event(rule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
		return nil
	}

	//Determine to update the rules in all patch bodies
	patchBody = as3.JudgeSelectedUpdate(adc, patchBody, isDelete)
	// find svc polices, if exists: svc policy have created
	jsonPath := strings.ReplaceAll(svcRouteDomainPolicePath, "/", ".")[1:]
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
				Path: fmt.Sprintf("%s/rules/-", svcRouteDomainPolicePath),
				Value: as3.Use{
					Use: as3Rule.Path,
				},
			}
			//if isDelete is true( if exist: remove );
			if isDelete {
				if index > -1 {
					policyItem.Op = as3.OpRemove
					policyItem.Path = fmt.Sprintf("%s/rules/%d", svcRouteDomainPolicePath, index)
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
			Rules: []as3.Use{
				//default fwr
				{Use: fmt.Sprintf(as3.GetAs3Config().ClusterName + as3.DenyAllRuleListName)},
			},
		}
		for _, as3Rule := range as3RulesList {
			policy.Rules = append(policy.Rules, as3.Use{Use: as3Rule.Path})
		}

		policyItem := as3.PatchItem{
			Path:  svcRouteDomainPolicePath,
			Op:    as3.OpAdd,
			Value: policy,
		}
		patchBody = append(patchBody, policyItem)

	}

	//vs policyFirewallEnforced point svc police
	vsPath := fmt.Sprintf("%s_outbound_vs", pathProfix)
	if !as3.GetAs3Config().IsSupportRouteDomain {
		//because only one vs
		vsPath = "/Common/Shared/k8s_outbound_vs"
	}
	jsonVsPath := strings.ReplaceAll(vsPath, "/", ".")[1:]
	result := gjson.Get(adc, jsonVsPath)
	if !result.Exists() {
		vs, err := as3.NewVirtualServer(nsConfig, false)
		if err != nil {
			klog.Errorf("NewVirtualServer failed: %v", err)
			return err
		}

		patchVsItem := as3.PatchItem{
			Op:    as3.OpAdd,
			Path:  vsPath,
			Value: vs,
		}

		gwPoll := as3.NewPoll(serverAddresses)

		patchPollItem := as3.PatchItem{
			Op:    as3.OpAdd,
			Path:  fmt.Sprintf("/%s/Shared/%s", tenant, vs.Pool),
			Value: gwPoll,
		}
		patchBody = append(patchBody, patchPollItem, patchVsItem)
		//patchBody = append(patchBody, )
	} else {
		res := result.Map()["policyFirewallEnforced"]
		if res.Exists() {
			if res.Map()["use"].String() != svcRouteDomainPolicePath {
				policeItem := as3.PatchItem{
					Op:   as3.OpReplace,
					Path: fmt.Sprintf("%s_outbound_vs/policyFirewallEnforced", pathProfix),
					Value: as3.Use{
						Use: svcRouteDomainPolicePath,
					},
				}
				patchBody = append(patchBody, policeItem)
			}
		} else {
			policeItem := as3.PatchItem{
				Op:   as3.OpAdd,
				Path: fmt.Sprintf("%s_outbound_vs/policyFirewallEnforced", pathProfix),
				Value: as3.Use{
					Use: svcRouteDomainPolicePath,
				},
			}
			patchBody = append(patchBody, policeItem)
		}
	}

	err = c.as3Client.Patch(patchBody...)
	if err != nil {
		err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
		klog.Error(err)
		return err
	}

	if !isDelete {
		rule.Status.Phase = kubeovn.ServiceEgressRuleSuccess
		_, err = c.as3clientset.KubeovnV1alpha1().ServiceEgressRules(namespace).UpdateStatus(context.Background(), rule, v1.UpdateOptions{})
		if err != nil {
			return err
		}
		c.recorder.Event(rule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}
	return nil
}
