package controller

import (
	"context"
	"fmt"

	kubeovn "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	}

	defer func() {
		if err != nil {
			c.recorder.Event(rule, corev1.EventTypeWarning, err.Error(), MessageResourceFailedSynced)
		}
	}()

	externalServicesList := kubeovn.ExternalServiceList{}
	for _, exsvcName := range rule.Spec.ExternalServices {
		exsvc, err := c.externalServicesLister.ExternalServices(as3.GetClusterSvcExtNamespace()).Get(exsvcName)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			klog.Warningf("externalService[%s/%s] does not exist", as3.GetClusterSvcExtNamespace(), exsvcName)
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
			if exsvc.Labels[as3.RuleTypeLabel] != as3.RuleTypeGlobal {
				exsvc.Labels[as3.RuleTypeLabel] = as3.RuleTypeGlobal
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
		klog.Warningf("ExternalServices is not found in clusterEgressRule[%s/%s], no need synchronize", rule.Namespace, rule.Name)
		return nil
	}
	if !isDelete && rule.Status.Phase != kubeovn.ClusterEgressRuleSyncing {
		rule.Status.Phase = kubeovn.ClusterEgressRuleSyncing
		rule, err = c.as3clientset.KubeovnV1alpha1().ClusterEgressRules().UpdateStatus(context.Background(), rule,
			metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	clusterEgressruleList := kubeovn.ClusterEgressRuleList{
		Items: []kubeovn.ClusterEgressRule{
			*rule,
		},
	}
	tntcfg := as3.GetTenantConfigForParttition(as3.DefaultPartition)
	err = c.as3Client.As3Request(nil, nil, &clusterEgressruleList, &externalServicesList, nil, nil, nil,
		tntcfg, as3.RuleTypeGlobal, isDelete)
	if err != nil {
		klog.Error(err)
		return err
	}

	if !isDelete {
		rule.Status.Phase = kubeovn.ClusterEgressRuleSuccess
		_, err = c.as3clientset.KubeovnV1alpha1().ClusterEgressRules().UpdateStatus(context.Background(), rule, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		c.recorder.Event(rule, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}
	return nil
}
