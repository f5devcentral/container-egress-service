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

	//tenant := nsConfig.Parttion
	//pathProfix := as3.AS3PathPrefix(nsConfig)
	//gw_pool.ServerAddresses
	//serverAddresses := nsConfig.Gwpool.ServerAddresses
	//routeDomain := nsConfig.RouteDomain

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

	exsvcs := make([]kubeovn.ExternalService, len(rule.Spec.ExternalServices))
	for i, svcName := range rule.Spec.ExternalServices {
		exsvcs[i] = kubeovn.ExternalService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: rule.Namespace,
			},
		}
	}

	eg := egress{
		name:rule.Name,
		namespace: rule.Namespace,
		exsvcs: exsvcs,
		action: rule.Spec.Action,
		ruleType: as3.RuleTypeService,
		k8sSvc: rule.Spec.Service,
		isDelete: isDelete,
	}

	_, patchBody, err := c.pkgEgress(eg, nsConfig)
	if err != nil{
		return err
	}

	//svcRouteDomainPolicePath := fmt.Sprintf("%s_svc_policy_%s", pathProfix, routeDomain.Name)
	//if !as3.GetAs3Config().IsSupportRouteDomain {
	//	//because only one svc police
	//	svcRouteDomainPolicePath = fmt.Sprintf("/Common/Shared/%s_svc_policy_rd", as3.GetAs3Config().ClusterName)
	//}
	//// get AS3 declaration
	//isExistTenant := true
	//adc, err := c.as3Client.Get(tenant)
	//if err != nil {
	//	if as3.IsNotFound(err) {
	//		isExistTenant = false
	//	} else {
	//		return fmt.Errorf("failed to get AS3: %v", err)
	//	}
	//}
	//
	////add tenant
	//if !isExistTenant {
	//	policy := as3.FirewallPolicy{
	//		Class: as3.ClassFirewallPolicy,
	//		Rules: []as3.Use{},
	//	}
	//	for _, as3Rule := range as3RulesList {
	//		policy.Rules = append(policy.Rules, as3.Use{Use: as3Rule.Path})
	//	}
	//	policyItem := as3.PatchItem{
	//		Path:  svcRouteDomainPolicePath,
	//		Op:    as3.OpAdd,
	//		Value: policy,
	//	}
	//	patchBody = append(patchBody, policyItem)
	//	as3Tenant, err := as3.NewAs3Tenant(nsConfig, patchBody)
	//	if err != nil {
	//		return err
	//	}
	//
	//	as3Tenant["defaultRouteDomain"] = routeDomain.Id
	//	tenantItem := as3.PatchItem{
	//		Op:    as3.OpAdd,
	//		Path:  "/" + tenant,
	//		Value: as3Tenant,
	//	}
	//
	//	//search route domian
	//	url := fmt.Sprintf("/mgmt/tm/net/route-domain/~%s~%s", tenant, routeDomain.Name)
	//	_, err = c.as3Client.GetF5Resource(url)
	//	if err != nil {
	//		klog.Errorf("failed to get route domian %s, error:%v", routeDomain.Name, err)
	//		return err
	//	}
	//
	//	err = c.as3Client.Patch(tenantItem)
	//	if err != nil {
	//		err = fmt.Errorf("failed to request AS3 Patch API: %v", err)
	//		klog.Error(err)
	//		return err
	//	}
	//	klog.Infof("as3 add %s tenant success", tenant)
	//	c.recorder.Event(rule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	//	return nil
	//}
	//
	////Determine to update the rules in all patch bodies
	//patchBody = as3.JudgeSelectedUpdate(adc, patchBody, isDelete)
	//// find svc polices, if exists: svc policy have created
	//jsonPath := strings.ReplaceAll(svcRouteDomainPolicePath, "/", ".")[1:]
	//if ok := gjson.Get(adc, jsonPath).Exists(); ok {
	//	for _, as3Rule := range as3RulesList {
	//		policyRuleList := gjson.Get(adc, fmt.Sprintf("%s.rules", jsonPath)).Array()
	//		//find index the value of item.Path
	//		index := -1
	//		for i, rule := range policyRuleList {
	//			if rule.Get("use").String() == as3Rule.Path {
	//				index = i
	//				break
	//			}
	//		}
	//		policyItem := as3.PatchItem{
	//			Path: fmt.Sprintf("%s/rules/-", svcRouteDomainPolicePath),
	//			Value: as3.Use{
	//				Use: as3Rule.Path,
	//			},
	//		}
	//		//if isDelete is true( if exist: remove );
	//		if isDelete {
	//			if index > -1 {
	//				policyItem.Op = as3.OpRemove
	//				policyItem.Path = fmt.Sprintf("%s/rules/%d", svcRouteDomainPolicePath, index)
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
	//		Rules: []as3.Use{
	//			//default fwr
	//			{Use: fmt.Sprintf(as3.GetAs3Config().ClusterName + as3.DenyAllRuleListName)},
	//		},
	//	}
	//	for _, as3Rule := range as3RulesList {
	//		policy.Rules = append(policy.Rules, as3.Use{Use: as3Rule.Path})
	//	}
	//
	//	policyItem := as3.PatchItem{
	//		Path:  svcRouteDomainPolicePath,
	//		Op:    as3.OpAdd,
	//		Value: policy,
	//	}
	//	patchBody = append(patchBody, policyItem)
	//}

	//vs policyFirewallEnforced point svc police
	//vsPath := fmt.Sprintf("%s_outbound_vs", pathProfix)
	//if !as3.GetAs3Config().IsSupportRouteDomain {
	//	//because only one vs
	//	vsPath = fmt.Sprintf("/Common/Shared/%s_outbound_vs", as3.GetAs3Config().ClusterName)
	//}
	//jsonVsPath := strings.ReplaceAll(vsPath, "/", ".")[1:]
	//result := gjson.Get(adc, jsonVsPath)
	//if !result.Exists() {
	//	vs, err := as3.NewVirtualServer(nsConfig)
	//	if err != nil {
	//		klog.Errorf("NewVirtualServer failed: %v", err)
	//		return err
	//	}
	//
	//	patchVsItem := as3.PatchItem{
	//		Op:    as3.OpAdd,
	//		Path:  vsPath,
	//		Value: vs,
	//	}
	//
	//	gwPoll := as3.NewPoll(serverAddresses)
	//
	//	patchPollItem := as3.PatchItem{
	//		Op:    as3.OpAdd,
	//		Path:  fmt.Sprintf("/%s/Shared/%s", tenant, vs.Pool),
	//		Value: gwPoll,
	//	}
	//	patchBody = append(patchBody, patchPollItem, patchVsItem)
	//	//patchBody = append(patchBody, )
	//} else {
	//	res := result.Map()["policyFirewallEnforced"]
	//	if res.Exists() {
	//		if res.Map()["use"].String() != svcRouteDomainPolicePath {
	//			policeItem := as3.PatchItem{
	//				Op:   as3.OpReplace,
	//				Path: fmt.Sprintf("%s_outbound_vs/policyFirewallEnforced", pathProfix),
	//				Value: as3.Use{
	//					Use: svcRouteDomainPolicePath,
	//				},
	//			}
	//			patchBody = append(patchBody, policeItem)
	//		}
	//	} else {
	//		policeItem := as3.PatchItem{
	//			Op:   as3.OpAdd,
	//			Path: fmt.Sprintf("%s_outbound_vs/policyFirewallEnforced", pathProfix),
	//			Value: as3.Use{
	//				Use: svcRouteDomainPolicePath,
	//			},
	//		}
	//		patchBody = append(patchBody, policeItem)
	//	}
	//}

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
