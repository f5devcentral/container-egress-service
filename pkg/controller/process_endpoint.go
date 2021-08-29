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
		klog.Infof("Successfully synced '%s'", key)
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

	nsConfig := as3.GetConfigNamespace(namespace)
	if nsConfig == nil {
		klog.Infof("namespace[%s] not in watch range ", namespace)
		return nil
	}

	klog.Infof("===============================>start sync endpoints[%s/%s]", namespace, name)
	defer klog.Infof("===============================>end sync endpoints[%s/%s]", namespace, name)

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
			// pathProfix := as3.AS3PathPrefix(as3.GetConfigNamespace(namespace))
			// srcAddrList := as3.FirewallAddressList{
			// 	Class: as3.ClassFirewallAddressList,
			// }

			// //get src ip
			// for _, subset := range ep.Subsets {
			// 	for _, addr := range subset.Addresses {
			// 		srcAddrList.Addresses = append(srcAddrList.Addresses, addr.IP)
			// 	}
			// }

			// patchBody := []as3.PatchItem{}
			// //update to ruleList source addr

			// patchItem := as3.PatchItem{
			// 	Path:  fmt.Sprintf("%s_svc_%s_%s_src_%s", pathProfix, rule.Namespace, rule.Name, ep.Name),
			// 	Op:    as3.OpReplace,
			// 	Value: srcAddrList,
			// }
			// patchBody = append(patchBody, patchItem)
			// if err = c.as3Client.Patch(patchBody...); err != nil {
			// 	err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
			// 	klog.Error(err)
			// 	return err
			// }

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

			if len(patchItems.Addresses) == 0{
				err = fmt.Errorf("endpoint[%s] subsets.addresses is nil", key)
				klog.Error(err)
				return err
			}
			url := fmt.Sprintf("/mgmt/tm/security/firewall/address-list/~%s~Shared~%s_svc_%s_%s_src_addr_%s", nsConfig.Parttion, as3.GetAs3Config().ClusterName, rule.Namespace, rule.Name, ep.Name)
			if nsConfig.RouteDomain.Id != 0 {
				for k := range patchItems.Addresses {
					patchItems.Addresses[k].Name = patchItems.Addresses[k].Name + "%10"
				}
			}

			err := c.as3Client.PatchF5Reource(patchItems, url)
			if err != nil {
				err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
				klog.Error(err)
				return err
			}
			break
		}
	}

	c.recorder.Event(endpoints, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}
