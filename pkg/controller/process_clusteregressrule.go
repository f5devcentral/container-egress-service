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

func (c *Controller) processNextClusterEgressRuleWorkerItem() bool {
	obj, shutdown := c.clusterEgressRuleWorkqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.clusterEgressRuleWorkqueue.Done(obj)

		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			c.clusterEgressRuleWorkqueue.Forget(obj)
			utilruntime.HandleError(err)
			return err
		}

		var rule *kubeovn.ClusterEgressRule
		var ok bool
		if rule, ok = obj.(*kubeovn.ClusterEgressRule); !ok {
			c.clusterEgressRuleWorkqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected clusterEgressRule in workqueue but got %#v", obj))
			return nil
		}

		if err := c.f5ClusterEgressRuleSyncHandler(key, rule); err != nil {
			c.clusterEgressRuleWorkqueue.AddRateLimited(rule)
			return fmt.Errorf("error syncing clusterEgressRule[%s]: %s, requeuing", key, err.Error())
		}

		c.clusterEgressRuleWorkqueue.Forget(obj)
		klog.Infof("Successfully synced clusterEgressRule[%s]", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) f5ClusterEgressRuleSyncHandler(key string, rule *kubeovn.ClusterEgressRule) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	klog.Infof("===============================>start sync clusterEgressRule[%s]", name)
	defer klog.Infof("===============================>end sync clusterEgressRule[%s]", name)

	var isDelete bool
	var r *kubeovn.ClusterEgressRule
	if r, err = c.clusterEgressRuleLister.Get(name); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		isDelete = true
	} else {
		rule = r
		if rule.Status.Phase != kubeovn.ClusterEgressRuleSyncing {
			rule.Status.Phase = kubeovn.ClusterEgressRuleSyncing
			rule, err = c.as3clientset.KubeovnV1alpha1().ClusterEgressRules().Update(context.Background(), rule,
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
		svc, err := c.externalServicesLister.ExternalServices(as3.ClusterSvcExtNamespace).Get(svcName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			klog.Warningf("external service %s does not exist", svcName)
			continue
		}

		//update ext ruleType global
		if svc.Labels == nil {
			svc.Labels = make(map[string]string, 1)
		}
		if svc.Labels[as3.RuleTypeLabel] != as3.RuleTypeGlobal {
			svc.Labels[as3.RuleTypeLabel] = as3.RuleTypeGlobal
			_, err = c.as3clientset.KubeovnV1alpha1().ExternalServices(as3.ClusterSvcExtNamespace).Update(context.Background(), svc, v1.UpdateOptions{})
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
			Path:  fmt.Sprintf("/Common/Shared/k8s_global_%s_ext_%s_address", rule.Name, svcName),
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
				Path:  fmt.Sprintf("/Common/Shared/k8s_global_%s_ext_%s_ports_%s", rule.Name, svcName, protocol),
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
			Path:  fmt.Sprintf("/Common/Shared/k8s_global_%s_ext_%s_rule_list", rule.Name, svcName),
			Op:    as3.OpAdd,
			Value: as3Rules,
		}
		as3RulesList = append(as3RulesList, as3RulesItem)
		patchBody = append(patchBody, addrItem, as3RulesItem)
		patchBody = append(patchBody, portList...)
	}

	// get AS3 declaration
	adc, err := c.as3Client.Get("Common")
	if err != nil {
		return fmt.Errorf("failed to get rule list index: %v", err)
	}

	//Determine to update the rules in all patch bodies
	patchBody = as3.JudgeSelectedUpdate(adc, patchBody, isDelete)

	// find global polices, if exists: global policy have created
	if ok := gjson.Get(adc, "Common.Shared.k8s_system_global_policy").Exists(); ok {
		for _, as3Rule := range as3RulesList {
			policyRuleList := gjson.Get(adc, "Common.Shared.k8s_system_global_policy.rules").Array()
			//find index the value of item.Path
			index := -1
			for i, rule := range policyRuleList {
				if rule.Get("use").String() == as3Rule.Path {
					index = i
					break
				}
			}
			policyItem := as3.PatchItem{
				Path: "/Common/Shared/k8s_system_global_policy/rules/-",
				Value: as3.Use{
					Use: as3Rule.Path,
				},
			}
			//if isDelete is true( if exist: remove );
			if isDelete {
				if index > -1 {
					policyItem.Op = as3.OpRemove
					policyItem.Path = fmt.Sprintf("/Common/Shared/k8s_system_global_policy/rules/%d", index)
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
			Path:  "/Common/Shared/k8s_system_global_policy",
			Op:    as3.OpAdd,
			Value: policy,
		}
		patchBody = append(patchBody, policyItem)

	}

	err = c.as3Client.Patch(patchBody...)
	if err != nil {
		err = fmt.Errorf("failed to request AS3 Patch API: %v", err)
		klog.Error(err)
		return err
	}

	//get route domian police
	url := "/mgmt/tm/security/firewall/global-rules"
	response, err := c.as3Client.GetF5Resource(url)
	if err != nil {
		return err
	}

	isExist := false
	if val, ok := response["enforcedPolicy"]; ok {
		fwEnforcedPolicy := val.(string)
		if fwEnforcedPolicy == "/Common/Shared/k8s_system_global_policy" {
			isExist = true
		}
	}

	// created global policy
	if !isExist {
		globalPolicy := map[string]string{
			"enforcedPolicy": "/Common/Shared/k8s_system_global_policy",
		}
		err := c.as3Client.PatchF5Reource(globalPolicy, url)
		if err != nil {
			return err
		}
	}

	if !isDelete {
		rule.Status.Phase = kubeovn.ClusterEgressRuleSuccess
		_, err = c.as3clientset.KubeovnV1alpha1().ClusterEgressRules().Update(context.Background(), rule, v1.UpdateOptions{})
		if err != nil {
			return err
		}
		c.recorder.Event(rule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}
	return nil
}
