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

	nsConfig := as3.GetTenantConfigForNamespace(namespace)
	if nsConfig == nil {
		klog.Infof("namespace[%s] not in watch range ", namespace)
		return nil
	}

	klog.Infof("===============================>start sync serviceEgressRule[%s/%s]", namespace, name)
	defer klog.Infof("===============================>end sync serviceEgressRule[%s/%s]", namespace, name)

	var isDelete bool
	var r *kubeovn.ServiceEgressRule
	if r, err = c.seviceEgressRuleLister.ServiceEgressRules(namespace).Get(name); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		isDelete = true
	} else {
		rule = r
	}

	defer func() {
		if err != nil {
			c.recorder.Event(rule, corev1.EventTypeWarning, err.Error(), MessageResourceFailedSynced)
		}
	}()

	ep, err := c.endpointsLister.Endpoints(namespace).Get(rule.Spec.Service)
	if err != nil {
		klog.Errorf("failed to get endpoint [%s/%s],due to: %v", namespace, rule.Spec.Service, err)
		return err
	}

	externalServicesList := kubeovn.ExternalServiceList{}
	//set source address, ns subnet

	endpointsList := corev1.EndpointsList{
		Items: []corev1.Endpoints{
			*ep,
		},
	}
	for _, exsvcName := range rule.Spec.ExternalServices {
		exsvc, err := c.externalServicesLister.ExternalServices(rule.Namespace).Get(exsvcName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			klog.Warningf("externalService[%s/%s] does not exist", rule.Namespace, exsvcName)
			continue
		}

		//update ext ruleType namespace
		if exsvc.Labels == nil {
			exsvc.Labels = make(map[string]string, 1)
		}

		//delete ruleType
		if isDelete {
			delete(exsvc.Labels, as3.RuleTypeLabel)
			_, err := c.as3clientset.KubeovnV1alpha1().ExternalServices(exsvc.Namespace).Update(context.Background(), exsvc,
				metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		} else {
			if exsvc.Labels[as3.RuleTypeLabel] != as3.RuleTypeService {
				exsvc.Labels[as3.RuleTypeLabel] = as3.RuleTypeService
				_, err := c.as3clientset.KubeovnV1alpha1().ExternalServices(exsvc.Namespace).Update(context.Background(), exsvc,
					metav1.UpdateOptions{})
				if err != nil {
					return err
				}
			}
		}
		externalServicesList.Items = append(externalServicesList.Items, *exsvc)
	}
	if len(externalServicesList.Items) == 0 {
		klog.Warningf("ExternalServices is not found in serviceEgressRules[%s/%s], no need synchronize", rule.Namespace, rule.Name)
		return nil
	}
	if !isDelete && rule.Status.Phase != kubeovn.ServiceEgressRuleSyncing {
		rule.Status.Phase = kubeovn.ServiceEgressRuleSyncing
		rule, err = c.as3clientset.KubeovnV1alpha1().ServiceEgressRules(namespace).UpdateStatus(context.Background(), rule,
			v1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	serviceEgressruleList := kubeovn.ServiceEgressRuleList{
		Items: []kubeovn.ServiceEgressRule{
			*rule,
		},
	}
	tntcfg := as3.GetTenantConfigForNamespace(namespace)
	err = c.as3Client.As3Request(&serviceEgressruleList, nil, nil, &externalServicesList, nil, &endpointsList, nil,
		tntcfg, as3.RuleTypeService, isDelete)
	if err != nil {
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
