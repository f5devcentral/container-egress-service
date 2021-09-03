package controller

import (
	"context"
	"fmt"

	kubeovn "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			rule, err = c.as3clientset.KubeovnV1alpha1().ClusterEgressRules().UpdateStatus(context.Background(), rule,
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

	exsvcs := make([]kubeovn.ExternalService, len(rule.Spec.ExternalServices))
	for i, svcName := range rule.Spec.ExternalServices {
		exsvcs[i] = kubeovn.ExternalService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: as3.ClusterSvcExtNamespace,
			},
		}
	}
	eg := egress{
		name:rule.Name,
		exsvcs: exsvcs,
		action: rule.Spec.Action,
		ruleType: as3.RuleTypeGlobal,
	}

	nsConfig := &as3.As3Namespace{
		Parttion: as3.CommonKey,
	}

	globalPolicyPath, patchBody, err := c.pkgEgress(eg, nsConfig)
	if err != nil{
		return err
	}

	//// get AS3 declaration
	//adc, err := c.as3Client.Get("Common")
	//if err != nil {
	//	return fmt.Errorf("failed to get rule list index: %v", err)
	//}
	//
	////Determine to update the rules in all patch bodies
	//patchBody = as3.JudgeSelectedUpdate(adc, patchBody, isDelete)
	//
	//// find global polices, if exists: global policy have created
	//if ok := gjson.Get(adc, fmt.Sprintf("Common.Shared.%s_system_global_policy", as3.GetAs3Config().ClusterName)).Exists(); ok {
	//	for _, as3Rule := range as3RulesList {
	//		policyRuleList := gjson.Get(adc, fmt.Sprintf("Common.Shared.%s_system_global_policy.rules", as3.GetAs3Config().ClusterName)).Array()
	//		//find index the value of item.Path
	//		index := -1
	//		for i, rule := range policyRuleList {
	//			if rule.Get("use").String() == as3Rule.Path {
	//				index = i
	//				break
	//			}
	//		}
	//		policyItem := as3.PatchItem{
	//			Path: fmt.Sprintf("/Common/Shared/%s_system_global_policy/rules/-", as3.GetAs3Config().ClusterName),
	//			Value: as3.Use{
	//				Use: as3Rule.Path,
	//			},
	//		}
	//		//if isDelete is true( if exist: remove );
	//		if isDelete {
	//			if index > -1 {
	//				policyItem.Op = as3.OpRemove
	//				policyItem.Path = fmt.Sprintf("/Common/Shared/%s_system_global_policy/rules/%d", as3.GetAs3Config().ClusterName, index)
	//				patchBody = append(patchBody, policyItem)
	//			}
	//		} else {
	//			//don,t exist: add
	//			if index == -1 {
	//				policyItem.Op = as3.OpAdd
	//				patchBody = append(patchBody, policyItem)
	//			}
	//		}
	//	}
	//} else {
	//	policy := as3.FirewallPolicy{
	//		Class: as3.ClassFirewallPolicy,
	//		Rules: []as3.Use{},
	//	}
	//	for _, as3Rule := range as3RulesList {
	//		policy.Rules = append(policy.Rules, as3.Use{Use: as3Rule.Path})
	//	}
	//	policyItem := as3.PatchItem{
	//		Path:  fmt.Sprintf("/Common/Shared/%s_system_global_policy", as3.GetAs3Config().ClusterName),
	//		Op:    as3.OpAdd,
	//		Value: policy,
	//	}
	//	patchBody = append(patchBody, policyItem)
	//
	//}

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
	if val, ok := response[as3.EnforcedPolicyKey]; ok {
		fwEnforcedPolicy := val.(string)
		if fwEnforcedPolicy == globalPolicyPath {
			isExist = true
		}
	}

	// created global policy
	if !isExist {
		globalPolicy := map[string]string{
			as3.EnforcedPolicyKey: globalPolicyPath,
		}
		err := c.as3Client.PatchF5Reource(globalPolicy, url)
		if err != nil {
			return err
		}

		err = c.as3Client.StoreDisk()
		if err !=nil {
			return err
		}
	}

	if !isDelete {
		rule.Status.Phase = kubeovn.ClusterEgressRuleSuccess
		_, err = c.as3clientset.KubeovnV1alpha1().ClusterEgressRules().UpdateStatus(context.Background(), rule, v1.UpdateOptions{})
		if err != nil {
			return err
		}
		c.recorder.Event(rule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}
	return nil
}
