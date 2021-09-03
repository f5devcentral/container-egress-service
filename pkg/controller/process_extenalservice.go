package controller

import (
	"fmt"
	kubeovn "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"strings"
)

func (c *Controller) processNextExternalServiceWorkItem() bool {
	obj, shutdown := c.externalServiceWorkqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.externalServiceWorkqueue.Done(obj)

		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			c.externalServiceWorkqueue.Forget(obj)
			utilruntime.HandleError(err)
			return err
		}

		var es *kubeovn.ExternalService
		var ok bool
		if es, ok = obj.(*kubeovn.ExternalService); !ok {
			c.externalServiceWorkqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected ExternalService in workqueue but got %#v", obj))
			return nil
		}

		if err := c.externalServiceSyncHandler(key, es); err != nil {
			c.externalServiceWorkqueue.AddRateLimited(es)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.externalServiceWorkqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) externalServiceSyncHandler(key string, service *kubeovn.ExternalService) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	klog.Infof("===============================>start sync externalService[%s]", name)
	defer klog.Infof("===============================>end sync externalService[%s]", name)

	var es *kubeovn.ExternalService
	if es, err = c.externalServicesLister.ExternalServices(namespace).Get(name); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		klog.Errorf("externalservices[%s] not found", name)
		return nil
	} else {
		service = es
	}

	defer func() {
		if err != nil {
			c.recorder.Event(service, corev1.EventTypeWarning, err.Error(), MessageResourceFailedSynced)
		}
	}()

	portMap := make(map[string][]string)
	for _, port := range service.Spec.Ports {
		if port.Port != "" {
			protocol := strings.ToLower(port.Protocol)
			portMap[protocol] = append(portMap[protocol], strings.Split(port.Port, ",")...)
		}
	}

	ruleType := es.Labels[as3.RuleTypeLabel]
	portLists := make([]as3.PatchItem, 0)
	addrList := as3.PatchItem{
		Op: as3.OpReplace,
		Value: as3.FirewallAddressList{
			Class:     as3.ClassFirewallAddressList,
			Addresses: service.Spec.Addresses,
		},
	}
	find := false
	switch ruleType {
	case as3.RuleTypeGlobal:
		ruleList, err := c.clusterEgressRuleLister.List(labels.Everything())
		if err != nil {
			return err
		}
		for _, rule := range ruleList {
			if find {
				break
			}
			for _, exSvc := range rule.Spec.ExternalServices {
				if exSvc == es.Name {
					find = true
					addrList.Path = fmt.Sprintf("/Common/Shared/%s_global_%s_ext_%s_address", as3.GetCluster(), rule.Name,
						service.Name)
					for protocol, ports := range portMap {
						if len(ports) != 0 {
							patchItem := as3.PatchItem{
								Op: as3.OpReplace,
								Value: as3.FirewallPortList{
									Class: as3.ClassFirewallPortList,
									Ports: ports,
								},
							}
							patchItem.Path = fmt.Sprintf("/Common/Shared/%s_global_%s_ext_%s_ports_%s", as3.GetCluster(),
								rule.Name, service.Name, protocol)
							portLists = append(portLists, patchItem)
						}
					}
					break
				}
			}
		}
	case as3.RuleTypeNamespace:
		ruleList, err := c.namespaceEgressRuleLister.List(labels.Everything())
		if err != nil {
			return err
		}
		for _, rule := range ruleList {
			if find {
				break
			}
			for _, exSvc := range rule.Spec.ExternalServices {
				if exSvc == es.Name {
					find = true
					pathProfix := as3.AS3PathPrefix(as3.GetConfigNamespace(rule.Namespace))
					addrList.Path = fmt.Sprintf("%s_ns_%s_%s_ext_%s_address", pathProfix, rule.Namespace, rule.Name, service.Name)
					for protocol, ports := range portMap {
						if len(ports) != 0 {
							patchItem := as3.PatchItem{
								Op: as3.OpReplace,
								Value: as3.FirewallPortList{
									Class: as3.ClassFirewallPortList,
									Ports: ports,
								},
							}

							patchItem.Path = fmt.Sprintf("%s_ns_%s_%s_ext_%s_ports_%s", pathProfix, rule.Namespace,
								rule.Name, service.Name, protocol)
							portLists = append(portLists, patchItem)
						}
					}
					break
				}
			}
		}
	case as3.RuleTypeService:
		ruleList, err := c.seviceEgressRuleLister.List(labels.Everything())
		if err != nil {
			return err
		}
		for _, rule := range ruleList {
			if find {
				break
			}
			for _, exSvc := range rule.Spec.ExternalServices {
				if exSvc == es.Name {
					pathProfix := as3.AS3PathPrefix(as3.GetConfigNamespace(rule.Namespace))
					find = true
					addrList.Path = fmt.Sprintf("%s_svc_%s_%s_ext_%s_address", pathProfix, rule.Namespace, rule.Name, service.Name)
					for protocol, ports := range portMap {
						if len(ports) != 0 {
							patchItem := as3.PatchItem{
								Op: as3.OpReplace,
								Value: as3.FirewallPortList{
									Class: as3.ClassFirewallPortList,
									Ports: ports,
								},
							}

							patchItem.Path = fmt.Sprintf("%s_svc_%s_%s_ext_%s_ports_%s", pathProfix, rule.Namespace, rule.Name,
								service.Name, protocol)
							portLists = append(portLists, patchItem)
						}
					}
					break
				}
			}
		}
	default:
		klog.Info("don,t neet sync!")
		return nil
	}

	if addrList.Path == "" {
		return nil
	}
	if err = c.as3Client.Patch(append(portLists, addrList)...); err != nil {
		err = fmt.Errorf("failed to request BIG-IP Patch API: %v", err)
		klog.Error(err)
		return err
	}

	c.recorder.Event(es, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}
