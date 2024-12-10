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

	nsConfig := as3.GetTenantConfigForNamespace(namespace)
	if nsConfig == nil {
		klog.Infof("namespace[%s] not in watch range ", namespace)
		return nil
	}

	klog.Infof("===============================>start sync namespaceEgressRule[%s/%s]", namespace, name)
	defer klog.Infof("===============================>end sync namespaceEgressRule[%s/%s]", namespace, name)

	var isDelete bool
	var r *kubeovn.NamespaceEgressRule
	if r, err = c.namespaceEgressRuleLister.NamespaceEgressRules(namespace).Get(name); err != nil {
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

	ns, err := c.kubeclientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get namespace[%s],due to: %v", namespace, err)
		return err
	}
	externalServicesList := kubeovn.ExternalServiceList{}
	//set source address, ns subnet

	namespaceList := corev1.NamespaceList{
		Items: []corev1.Namespace{
			*ns,
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
			if exsvc.Labels[as3.RuleTypeLabel] != as3.RuleTypeNamespace {
				exsvc.Labels[as3.RuleTypeLabel] = as3.RuleTypeNamespace
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
		klog.Warningf("ExternalServices is not found in namespaceEgressRules[%s/%s], no need synchronize", rule.Namespace, rule.Name)
		return nil
	}
	if !isDelete && rule.Status.Phase != kubeovn.NamespaceEgressRuleSyncing {
		rule.Status.Phase = kubeovn.NamespaceEgressRuleSyncing
		rule, err = c.as3clientset.KubeovnV1alpha1().NamespaceEgressRules(namespace).UpdateStatus(context.Background(), rule,
			v1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	namespaceEgressruleList := kubeovn.NamespaceEgressRuleList{
		Items: []kubeovn.NamespaceEgressRule{
			*rule,
		},
	}
	tntcfg := as3.GetTenantConfigForNamespace(namespace)
	err = c.as3Client.As3Request(nil, &namespaceEgressruleList, nil, &externalServicesList, nil, nil, &namespaceList,
		tntcfg, as3.RuleTypeNamespace, isDelete)
	if err != nil {
		klog.Error(err)
		return err
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
