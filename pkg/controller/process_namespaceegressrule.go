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

func (c *Controller) processNextNamespaceEgressRuleWorkerItem() bool {
	obj, shutdown := c.namespaceEgressRuleWorkqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.namespaceEgressRuleWorkqueue.Done(obj)

		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			c.namespaceEgressRuleWorkqueue.Forget(obj)
			utilruntime.HandleError(err)
			return err
		}

		var rule *kubeovn.NamespaceEgressRule
		var ok bool
		if rule, ok = obj.(*kubeovn.NamespaceEgressRule); !ok {
			c.namespaceEgressRuleWorkqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected NamespaceEgressRule in workqueue but got %#v", obj))
			return nil
		}

		if err := c.namespaceEgressRuleSyncHandler(key, rule); err != nil {
			c.namespaceEgressRuleWorkqueue.AddRateLimited(rule)
			return fmt.Errorf("error syncing namespaceEgressRule[%s]: %s, requeuing", key, err.Error())
		}

		c.namespaceEgressRuleWorkqueue.Forget(obj)
		klog.Infof("Successfully synced namespaceEgressRule[%s]", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) namespaceEgressRuleSyncHandler(key string, rule *kubeovn.NamespaceEgressRule) error {
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

	klog.Infof("===============================>start sync namespaceEgressRule[%s/%s]", namespace, name)
	defer klog.Infof("===============================>end sync namespaceEgressRule[%s/%s]", namespace, name)

	tenant := nsConfig.Parttion
	pathProfix := as3.AS3PathPrefix(nsConfig)
	routeDomain := nsConfig.RouteDomain
	var isDelete bool
	var r *kubeovn.NamespaceEgressRule
	if r, err = c.namespaceEgressRuleLister.NamespaceEgressRules(namespace).Get(name); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		isDelete = true
	} else {
		rule = r
		if rule.Status.Phase != kubeovn.NamespaceEgressRuleSyncing {
			rule.Status.Phase = kubeovn.NamespaceEgressRuleSyncing
			rule, err = c.as3clientset.KubeovnV1alpha1().NamespaceEgressRules(namespace).UpdateStatus(context.Background(), rule,
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

		//update ext ruleType namespace
		if svc.Labels == nil {
			svc.Labels = make(map[string]string, 1)
		}

		if svc.Labels[as3.RuleTypeLabel] != as3.RuleTypeNamespace {
			svc.Labels[as3.RuleTypeLabel] = as3.RuleTypeNamespace
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
			Path:  fmt.Sprintf("%s_ns_%s_%s_ext_%s_address", pathProfix, rule.Namespace, rule.Name, svcName),
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
				Path:  fmt.Sprintf("%s_ns_%s_%s_ext_%s_ports_%s", pathProfix, rule.Namespace, rule.Name, svcName, protocol),
				Op:    as3.OpAdd,
				Value: value,
			}
			portList = append(portList, portItem)
			//FirewallRule
			//global police don't have source
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
			}
			as3Rules.Rules = append(as3Rules.Rules, as3Rule)
		}
		as3RulesItem := as3.PatchItem{
			Path:  fmt.Sprintf("%s_ns_%s_%s_ext_%s_rule_list", pathProfix, rule.Namespace, rule.Name, svcName),
			Op:    as3.OpAdd,
			Value: as3Rules,
		}
		as3RulesList = append(as3RulesList, as3RulesItem)
		patchBody = append(patchBody, addrItem, as3RulesItem)
		patchBody = append(patchBody, portList...)
	}

	// get AS3 declaration
	isExistTenant := true
	adc, err := c.as3Client.Get(tenant)
	if err != nil {
		if as3.IsNotFound(err) {
			isExistTenant = false
		} else {
			return fmt.Errorf("failed to get BIG-IP: %v", err)
		}
	}
	nsRouteDomainPolicePath := fmt.Sprintf("%s_ns_policy_%s", pathProfix, routeDomain.Name)
	if !as3.GetAs3Config().IsSupportRouteDomain {
		//because only one ns police
		nsRouteDomainPolicePath = "/Common/Shared/k8s_ns_policy_rd"
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
			Path:  nsRouteDomainPolicePath,
			Op:    as3.OpAdd,
			Value: nsPolicy,
		}

		//add deny all policy
		svcPolicyItem := as3.PatchItem{
			Path: fmt.Sprintf("%s_svc_policy_%s", pathProfix, nsConfig.RouteDomain.Name),
			Op:   as3.OpAdd,
			Value: as3.FirewallPolicy{
				Class: as3.ClassFirewallPolicy,
				Rules: []as3.Use{},
			},
		}

		patchBody = append(patchBody, policyItem, svcPolicyItem)
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

		err = c.as3Client.Patch(tenantItem)
		if err != nil {
			err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
			klog.Error(err)
			return err
		}
		klog.Infof("BIG-IP add %s tenant success", tenant)
	}

	//Determine to update the rules in all patch bodies
	patchBody = as3.JudgeSelectedUpdate(adc, patchBody, isDelete)

	// find namespace polices, if exists: namespace policy have created

	jsonPath := strings.ReplaceAll(nsRouteDomainPolicePath, "/", ".")[1:]
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
				Path: fmt.Sprintf("%s/rules/-", nsRouteDomainPolicePath),
				Value: as3.Use{
					Use: as3Rule.Path,
				},
			}
			//if isDelete is true( if exist: remove );
			if isDelete {
				if index > -1 {
					policyItem.Op = as3.OpRemove
					policyItem.Path = fmt.Sprintf("%s/rules/%d", nsRouteDomainPolicePath, index)
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
			Path:  nsRouteDomainPolicePath,
			Op:    as3.OpAdd,
			Value: policy,
		}
		patchBody = append(patchBody, policyItem)

	}

	err = c.as3Client.Patch(patchBody...)
	if err != nil {
		err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
		klog.Error(err)
		return err
	}

	//get route domian police
	url := fmt.Sprintf("/mgmt/tm/net/route-domain/~%s~%s", tenant, routeDomain.Name)
	response, err := c.as3Client.GetF5Resource(url)
	if err != nil {
		klog.Errorf("failed to get route domian %s, error:%v", routeDomain.Name, err)
		return err
	}

	isNsPolicyExist := false
	if val, ok := response["fwEnforcedPolicy"]; ok {
		fwEnforcedPolicy := val.(string)
		if fwEnforcedPolicy == nsRouteDomainPolicePath {
			isNsPolicyExist = true
		}
	}
	if !isNsPolicyExist {
		// binding route domain policy
		rd := as3.RouteDomain{
			FwEnforcedPolicy: nsRouteDomainPolicePath,
		}
		err := c.as3Client.PatchF5Reource(rd, url)
		if err != nil {
			return err
		}
	}

	if !isDelete {
		rule.Status.Phase = kubeovn.NamespaceEgressRuleSuccess
		_, err = c.as3clientset.KubeovnV1alpha1().NamespaceEgressRules(namespace).UpdateStatus(context.Background(), rule, v1.UpdateOptions{})
		if err != nil {
			return err
		}
		c.recorder.Event(rule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}
	return nil
}
