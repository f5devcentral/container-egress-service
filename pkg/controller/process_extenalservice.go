package controller

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"

	kubeovn "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
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

	var isDelete bool
	var es *kubeovn.ExternalService
	if es, err = c.externalServicesLister.ExternalServices(namespace).Get(name); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		isDelete = true
		err = nil
	} else {
		service = es
	}

	defer func() {
		if err != nil {
			c.recorder.Event(service, corev1.EventTypeWarning, err.Error(), MessageResourceFailedSynced)
		}
	}()
	//verify bandwidth
	if !verifyExtenalService(service) {
		err = fmt.Errorf("The bandwidth field is invalid, one of them should be filled in %s", as3.GetIRules())
		return err
	}
	ruleType := service.Labels[as3.RuleTypeLabel]
	find := false
	clusterEgressruleList := kubeovn.ClusterEgressRuleList{}
	namespaceEgressRuleList := kubeovn.NamespaceEgressRuleList{}
	serviceEgressRuleList := kubeovn.ServiceEgressRuleList{}
	externalServicesList := kubeovn.ExternalServiceList{
		Items: []kubeovn.ExternalService{
			*service,
		},
	}
	namespaceList := corev1.NamespaceList{}
	endpointList := corev1.EndpointsList{}
	tntcfg := &as3.TenantConfig{}
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
				if exSvc == service.Name {
					find = true
					clusterEgressruleList.Items = append(clusterEgressruleList.Items, *rule)
					break
				}
			}
		}
		tntcfg = as3.GetTenantConfigForParttition(as3.DefaultPartition)
	case as3.RuleTypeNamespace:
		ruleList, err := c.namespaceEgressRuleLister.NamespaceEgressRules(service.Namespace).List(labels.Everything())
		if err != nil {
			return err
		}
		for _, rule := range ruleList {
			if find {
				break
			}
			for _, exSvc := range rule.Spec.ExternalServices {
				if exSvc == service.Name {
					find = true
					namespaceEgressRuleList.Items = append(namespaceEgressRuleList.Items, *rule)
					break
				}
			}
		}
		ns, err := c.kubeclientset.CoreV1().Namespaces().Get(context.Background(), service.Namespace, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get namespace[%s],due to: %v", service.Namespace, err)
			return err
		}
		namespaceList.Items = []corev1.Namespace{
			*ns,
		}
		tntcfg = as3.GetTenantConfigForNamespace(service.Namespace)
	case as3.RuleTypeService:
		ruleList, err := c.seviceEgressRuleLister.ServiceEgressRules(service.Namespace).List(labels.Everything())
		if err != nil {
			return err
		}
		for _, rule := range ruleList {
			if find {
				break
			}
			for _, exSvc := range rule.Spec.ExternalServices {
				if exSvc == service.Name {
					find = true
					serviceEgressRuleList.Items = append(serviceEgressRuleList.Items, *rule)
					break
				}
			}
		}
		if find {
			epName := serviceEgressRuleList.Items[0].Spec.Service
			ep, err := c.endpointsLister.Endpoints(service.Namespace).Get(epName)
			if err != nil {
				klog.Errorf("failed to get endpoint [%s/%s],due to: %v", service.Namespace, epName, err)
				return err
			}
			endpointList.Items = []corev1.Endpoints{
				*ep,
			}
		}
		tntcfg = as3.GetTenantConfigForNamespace(service.Namespace)
	default:
		klog.Info("don,t neet sync!")
		return nil
	}

	if len(serviceEgressRuleList.Items) == 0 && len(namespaceEgressRuleList.Items) == 0 && len(clusterEgressruleList.Items) == 0 {
		klog.Info("not found Associated rules，don,t neet sync!!")
		return nil
	}
	err = c.as3Client.As3Request(&serviceEgressRuleList, &namespaceEgressRuleList, &clusterEgressruleList, &externalServicesList, nil, &endpointList, &namespaceList,
		tntcfg, ruleType, isDelete)
	if err != nil {
		klog.Error(err)
		return err
	}
	c.recorder.Event(service, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func verifyExtenalService(exsvc *kubeovn.ExternalService) bool {
	ports := exsvc.Spec.Ports
	iruleStr := as3.GetIRules()
	for _, port := range ports {
		bindwidth := port.Bandwidth
		if strings.TrimSpace(bindwidth) != "" {
			if !strings.Contains(iruleStr, bindwidth) {
				return false
			}
		}
	}
	return true
}
