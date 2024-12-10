package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	snat "github.com/kubeovn/ces-controller/pkg/apis/bigip.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
)

func (c *Controller) processNextExternalIPRuleWorkItem() bool {
	obj, shutdown := c.externalIPRuleWorkQueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.externalIPRuleWorkQueue.Done(obj)

		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			c.externalIPRuleWorkQueue.Forget(obj)
			utilruntime.HandleError(err)
			return err
		}

		var eipRule *snat.ExternalIPRule
		var ok bool
		if eipRule, ok = obj.(*snat.ExternalIPRule); !ok {
			c.externalIPRuleWorkQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected ExternalIPRule in workqueue but got %#v", obj))
			return nil
		}

		if err := c.externalIPRuleSyncHandler(key, eipRule); err != nil {
			c.externalIPRuleWorkQueue.AddRateLimited(eipRule)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.externalIPRuleWorkQueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) externalIPRuleSyncHandler(key string, eipRule *snat.ExternalIPRule) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	klog.Infof("===============================>start sync externalIPRule[%s]", name)
	defer klog.Infof("===============================>end sync externalIPRule[%s]", name)

	var isDelete bool
	var rule *snat.ExternalIPRule
	if rule, err = c.externalIPRuleLister.ExternalIPRules(namespace).Get(name); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		isDelete = true
		err = nil
	} else {
		eipRule = rule
	}

	defer func() {
		if err != nil {
			c.recorder.Event(eipRule, corev1.EventTypeWarning, err.Error(), MessageResourceFailedSynced)
		}
	}()

	endpointList := &corev1.EndpointsList{
		Items: make([]corev1.Endpoints, 0, len(eipRule.Spec.Services)),
	}
	for _, svcName := range eipRule.Spec.Services {
		ep, err := c.endpointsLister.Endpoints(eipRule.Namespace).Get(svcName)
		if err != nil {
			klog.Errorf("get endpoint %s/%s error: %s", eipRule.Namespace, svcName, err.Error())
			continue
		}

		epCopy := ep.DeepCopy()
		endpointList.Items = append(endpointList.Items, *epCopy)
	}

	eipRuleCopy := eipRule.DeepCopy()
	eipRuleList := &snat.ExternalIPRuleList{
		Items: []snat.ExternalIPRule{
			*eipRuleCopy,
		},
	}

	tntcfg := as3.GetTenantConfigForNamespace(namespace)
	err = c.as3Client.As3Request(nil, nil, nil, nil, eipRuleList, endpointList, nil,
		tntcfg, "", isDelete)
	if err != nil {
		klog.Error(err)
		return err
	}

	c.recorder.Event(eipRule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// getEndpointIPsWithExternalIPRule 根据externalIPRule获取所有endpoint ip
func (c *Controller) getEndpointIPsWithExternalIPRule(eipRule *snat.ExternalIPRule) []string {
	var ips []string
	for _, svcName := range eipRule.Spec.Services {
		// svcName和epName相同
		ep, err := c.endpointsLister.Endpoints(eipRule.Namespace).Get(svcName)
		if err != nil {
			// 这里报错一般是不存在，忽略即可
			continue
		}

		for _, subnet := range ep.Subsets {
			for _, addr := range subnet.Addresses {
				ips = append(ips, addr.IP)
			}
		}
	}
	return ips
}
