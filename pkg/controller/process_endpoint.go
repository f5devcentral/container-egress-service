package controller

import (
	"fmt"

	"github.com/kubeovn/ces-controller/pkg/as3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func (c *Controller) processNextEndpointsWorkItem() bool {
	obj, shutdown := c.endpointsWorkqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.endpointsWorkqueue.Done(obj)

		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			c.endpointsWorkqueue.Forget(obj)
			utilruntime.HandleError(err)
			return err
		}

		var ep *corev1.Endpoints
		var ok bool
		if ep, ok = obj.(*corev1.Endpoints); !ok {
			c.endpointsWorkqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected Endpoints in workqueue but got %#v", obj))
			return nil
		}

		if err := c.endpointsSyncHandler(key, ep); err != nil {
			c.endpointsWorkqueue.AddRateLimited(ep)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.endpointsWorkqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) endpointsSyncHandler(key string, endpoints *corev1.Endpoints) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	nsConfig := as3.GetTenantConfigForNamespace(namespace)
	if nsConfig == nil {
		klog.Infof("namespace[%s] not in watch range ", namespace)
		return nil
	}

	ep, err := c.endpointsLister.Endpoints(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Errorf("endpoint [%s/%s] not found", namespace, name)
			return nil
		}
		klog.Errorf("failed to get endpoint [%s/%s],due to: %v", ep.Namespace, ep.Name, err)
		return err
	}

	defer func() {
		if err != nil {
			c.recorder.Event(ep, corev1.EventTypeWarning, err.Error(), MessageResourceFailedSynced)
		}
	}()

	as3Rules, err := c.seviceEgressRuleLister.ServiceEgressRules(namespace).List(labels.Everything())

	if err != nil {
		klog.Errorf("failed to list BIG-IP service egress rules: %v", err)
		return err
	}

	nameInRule := name

	for _, rule := range as3Rules {
		if rule.Spec.Service == nameInRule {
			//Due to frequent ip update,so BIG-IP native interface is used
			patchItems := as3.BigIpAddressList{}
			//get src ip
			for _, subset := range ep.Subsets {
				for _, addr := range subset.Addresses {
					patchItem := as3.BigIpAddresses{
						Name: addr.IP,
					}
					patchItems.Addresses = append(patchItems.Addresses, patchItem)
				}
			}

			if len(patchItems.Addresses) == 0 {
				err = fmt.Errorf("endpoint[%s] subsets.addresses is nil", key)
				klog.Error(err)
				return err
			}
			klog.Infof("===============================>start sync endpoints[%s/%s]", namespace, name)
			c.as3Client.UpdateBigIPSourceAddress(patchItems, nsConfig, namespace, rule.Name, ep.Name)
			defer klog.Infof("===============================>end sync endpoints[%s/%s]", namespace, name)
			break
		}
	}
	return nil
}
