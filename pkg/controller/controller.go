/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	snat "github.com/kubeovn/ces-controller/pkg/apis/bigip.io/v1alpha1"
	kubeovn "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/as3"
	clientset "github.com/kubeovn/ces-controller/pkg/generated/clientset/versioned"
	as3scheme "github.com/kubeovn/ces-controller/pkg/generated/clientset/versioned/scheme"
	snatinformers "github.com/kubeovn/ces-controller/pkg/generated/informers/externalversions/bigip.io/v1alpha1"
	informers "github.com/kubeovn/ces-controller/pkg/generated/informers/externalversions/kubeovn.io/v1alpha1"
	snatlisters "github.com/kubeovn/ces-controller/pkg/generated/listers/bigip.io/v1alpha1"
	listers "github.com/kubeovn/ces-controller/pkg/generated/listers/kubeovn.io/v1alpha1"
)

const ControllerAgentName = "ces-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a resource is synced
	SuccessSynced = "Synced"

	FailedSynced = "Failed Synced"

	// MessageResourceSynced is the message used for an Event fired when a resource
	// is synced successfully
	MessageResourceSynced = "synced successfully"

	MessageResourceFailedSynced = "synced Failed"
)

// Controller is the controller implementation for related resources
type Controller struct {
	kubeclientset kubernetes.Interface
	as3clientset  clientset.Interface

	endpointsLister              listersv1.EndpointsLister
	endpointsSynced              cache.InformerSynced
	endpointsWorkqueue           workqueue.RateLimitingInterface
	externalServicesLister       listers.ExternalServiceLister
	externalServicesSynced       cache.InformerSynced
	externalServiceWorkqueue     workqueue.RateLimitingInterface
	clusterEgressRuleLister      listers.ClusterEgressRuleLister
	clusterEgressRuleSynced      cache.InformerSynced
	clusterEgressRuleWorkqueue   workqueue.RateLimitingInterface
	namespaceEgressRuleLister    listers.NamespaceEgressRuleLister
	namespaceEgressRuleSynced    cache.InformerSynced
	namespaceEgressRuleWorkqueue workqueue.RateLimitingInterface
	seviceEgressRuleLister       listers.ServiceEgressRuleLister
	seviceEgressRuleSynced       cache.InformerSynced
	seviceEgressRuleWorkqueue    workqueue.RateLimitingInterface
	recorder                     record.EventRecorder
	as3Client                    *as3.Client

	// snat相关
	externalIPRuleLister    snatlisters.ExternalIPRuleLister
	externalIPRuleSynced    cache.InformerSynced
	externalIPRuleWorkQueue workqueue.RateLimitingInterface
}

// NewController returns a new CES controller
func NewController(
	kubeclientset kubernetes.Interface,
	as3clientset clientset.Interface,
	endpointsInformer kubeinformers.EndpointsInformer,
	externalServiceInformer informers.ExternalServiceInformer,
	clusterEgressRuleInformer informers.ClusterEgressRuleInformer,
	namespaceEgressRuleInformer informers.NamespaceEgressRuleInformer,
	seviceEgressRuleInformer informers.ServiceEgressRuleInformer,
	externalIPRuleInformer snatinformers.ExternalIPRuleInformer,
	as3Client *as3.Client) *Controller {

	utilruntime.Must(as3scheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: ControllerAgentName})

	controller := &Controller{
		kubeclientset:                kubeclientset,
		as3clientset:                 as3clientset,
		endpointsLister:              endpointsInformer.Lister(),
		endpointsSynced:              endpointsInformer.Informer().HasSynced,
		endpointsWorkqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
		externalServicesLister:       externalServiceInformer.Lister(),
		externalServicesSynced:       externalServiceInformer.Informer().HasSynced,
		externalServiceWorkqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ExternalServices"),
		clusterEgressRuleLister:      clusterEgressRuleInformer.Lister(),
		clusterEgressRuleSynced:      clusterEgressRuleInformer.Informer().HasSynced,
		clusterEgressRuleWorkqueue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ClusterEgressRules"),
		namespaceEgressRuleLister:    namespaceEgressRuleInformer.Lister(),
		namespaceEgressRuleSynced:    namespaceEgressRuleInformer.Informer().HasSynced,
		namespaceEgressRuleWorkqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NamespaceEgressRules"),
		seviceEgressRuleLister:       seviceEgressRuleInformer.Lister(),
		seviceEgressRuleSynced:       seviceEgressRuleInformer.Informer().HasSynced,
		seviceEgressRuleWorkqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "SeviceEgressRules"),
		recorder:                     recorder,
		as3Client:                    as3Client,

		externalIPRuleLister:    externalIPRuleInformer.Lister(),
		externalIPRuleSynced:    externalIPRuleInformer.Informer().HasSynced,
		externalIPRuleWorkQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ExternalIPRules"),
	}

	klog.Info("Setting up event handlers")

	endpointsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		//AddFunc: controller.enqueueEndpoints,
		UpdateFunc: func(old, new interface{}) {
			if !controller.isUpdate(old, new) {
				return
			}
			controller.enqueueEndpoints(new)
		},
		DeleteFunc: controller.enqueueEndpoints,
	})

	externalServiceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		//AddFunc: controller.enqueueExternalService,
		UpdateFunc: func(old, new interface{}) {
			if !controller.isUpdate(old, new) {

				return
			}
			controller.enqueueExternalService(new)
		},
		DeleteFunc: controller.enqueueExternalService,
	})

	clusterEgressRuleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueClusterEgressRule,
		UpdateFunc: func(old, new interface{}) {
			if !controller.isUpdate(old, new) {
				return
			}
			controller.enqueueClusterEgressRule(new)
		},
		DeleteFunc: controller.enqueueClusterEgressRule,
	})
	namespaceEgressRuleInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: controller.enqueueNamespaceEgressRule,
			UpdateFunc: func(old, new interface{}) {
				if !controller.isUpdate(old, new) {
					return
				}
				controller.enqueueNamespaceEgressRule(new)
			},
			DeleteFunc: controller.enqueueNamespaceEgressRule,
		})

	seviceEgressRuleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSeviceEgressRule,
		UpdateFunc: func(old, new interface{}) {
			if !controller.isUpdate(old, new) {
				return
			}
			controller.enqueueSeviceEgressRule(new)
		},
		DeleteFunc: controller.enqueueSeviceEgressRule,
	})

	externalIPRuleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueExternalIPRule,
		UpdateFunc: func(oldObj, newObj interface{}) {
			if !controller.isUpdate(oldObj, newObj) {
				return
			}
			controller.enqueueExternalIPRule(newObj)
		},
		DeleteFunc: controller.enqueueExternalIPRule,
	})

	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.endpointsWorkqueue.ShutDown()
	defer c.externalServiceWorkqueue.ShutDown()
	defer c.clusterEgressRuleWorkqueue.ShutDown()
	defer c.namespaceEgressRuleWorkqueue.ShutDown()
	defer c.seviceEgressRuleWorkqueue.ShutDown()
	defer c.externalIPRuleWorkQueue.ShutDown()

	klog.Info("Starting CES controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.endpointsSynced); !ok {
		return fmt.Errorf("failed to wait for endpoints caches to sync")
	}
	if ok := cache.WaitForCacheSync(stopCh, c.externalServicesSynced); !ok {
		return fmt.Errorf("failed to wait for external service caches to sync")
	}
	if ok := cache.WaitForCacheSync(stopCh, c.clusterEgressRuleSynced); !ok {
		return fmt.Errorf("failed to wait for BIG-IP cluster egress rule caches to sync")
	}
	if ok := cache.WaitForCacheSync(stopCh, c.namespaceEgressRuleSynced); !ok {
		return fmt.Errorf("failed to wait for BIG-IP namespace egress rule caches to sync")
	}
	if ok := cache.WaitForCacheSync(stopCh, c.seviceEgressRuleSynced); !ok {
		return fmt.Errorf("failed to wait for BIG-IP service egress rule caches to sync")
	}
	if ok := cache.WaitForCacheSync(stopCh, c.externalIPRuleSynced); !ok {
		return fmt.Errorf("failed to wait for snat external ip rule caches to sync")
	}

	klog.Info("Starting workers")
	go wait.Until(c.runEndpointsWorker, 5*time.Second, stopCh)
	go wait.Until(c.runExternalServiceWorker, 5*time.Second, stopCh)
	go wait.Until(c.runClusterEgressRuleWorker, 5*time.Second, stopCh)
	go wait.Until(c.runNamespaceEgressRuleWorker, 5*time.Second, stopCh)
	go wait.Until(c.runSeviceEgressRuleWorker, 5*time.Second, stopCh)
	go wait.Until(c.runExternalIPRuleWorker, 5*time.Second, stopCh)

	klog.Info("Started workers")
	for i := 0; i < 5; i++ {
		<-stopCh
	}

	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runEndpointsWorker() {
	for c.processNextEndpointsWorkItem() {
	}
}

func (c *Controller) runExternalServiceWorker() {
	for c.processNextExternalServiceWorkItem() {
	}
}

func (c *Controller) runClusterEgressRuleWorker() {
	for c.processNextClusterEgressRuleWorkerItem() {
	}
}

func (c *Controller) runNamespaceEgressRuleWorker() {
	for c.processNextNamespaceEgressRuleWorkerItem() {
	}
}

func (c *Controller) runSeviceEgressRuleWorker() {
	for c.processNextSeviceEgressRuleWorkerItem() {
	}
}

func (c *Controller) runExternalIPRuleWorker() {
	for c.processNextExternalIPRuleWorkItem() {
	}
}

func (c *Controller) enqueueEndpoints(obj interface{}) {
	c.endpointsWorkqueue.Add(obj)
}

func (c *Controller) enqueueExternalService(obj interface{}) {
	c.externalServiceWorkqueue.Add(obj)
}

func (c *Controller) enqueueClusterEgressRule(obj interface{}) {
	c.clusterEgressRuleWorkqueue.Add(obj)
}

func (c *Controller) enqueueNamespaceEgressRule(obj interface{}) {
	c.namespaceEgressRuleWorkqueue.Add(obj)
}

func (c *Controller) enqueueSeviceEgressRule(obj interface{}) {
	c.seviceEgressRuleWorkqueue.Add(obj)
}

func (c *Controller) enqueueExternalIPRule(obj interface{}) {
	c.externalIPRuleWorkQueue.Add(obj)
}

func (c *Controller) isUpdate(old, new interface{}) bool {
	switch old.(type) {
	case *kubeovn.ClusterEgressRule:
		oldRule := old.(*kubeovn.ClusterEgressRule)
		newRule := new.(*kubeovn.ClusterEgressRule)

		if oldRule.ResourceVersion == newRule.ResourceVersion {
			return false
		}
		if oldRule.Spec.Action != newRule.Spec.Action {
			return true
		}
		if oldRule.Spec.Logging != newRule.Spec.Logging {
			return true
		}
		if !reflect.DeepEqual(oldRule.Spec.ExternalServices, newRule.Spec.ExternalServices) {
			return true
		}
	case *kubeovn.NamespaceEgressRule:
		oldNsRule := old.(*kubeovn.NamespaceEgressRule)
		newNsRule := new.(*kubeovn.NamespaceEgressRule)
		nsConfig := as3.GetTenantConfigForNamespace(oldNsRule.Namespace)
		if nsConfig == nil {
			klog.Infof("namespace[%s] not in watch range ", oldNsRule.Namespace)
			return false
		}

		if oldNsRule.ResourceVersion == newNsRule.ResourceVersion {
			return false
		}
		if oldNsRule.Spec.Action != newNsRule.Spec.Action {
			return true
		}
		if oldNsRule.Spec.Logging != newNsRule.Spec.Logging {
			return true
		}
		if !reflect.DeepEqual(oldNsRule.Spec.ExternalServices, newNsRule.Spec.ExternalServices) {
			return true
		}
	case *kubeovn.ServiceEgressRule:
		oldSvcRule := old.(*kubeovn.ServiceEgressRule)
		newSvcRule := new.(*kubeovn.ServiceEgressRule)
		nsConfig := as3.GetTenantConfigForNamespace(oldSvcRule.Namespace)
		if nsConfig == nil {
			klog.Infof("namespace[%s] not in watch range ", oldSvcRule.Namespace)
			return false
		}
		if oldSvcRule.ResourceVersion == newSvcRule.ResourceVersion {
			return false
		}
		if oldSvcRule.Spec.Action != newSvcRule.Spec.Action {
			return true
		}
		if oldSvcRule.Spec.Logging != newSvcRule.Spec.Logging {
			return true
		}
		if oldSvcRule.Spec.Service != oldSvcRule.Spec.Service {
			return true
		}
		if !reflect.DeepEqual(oldSvcRule.Spec.ExternalServices, newSvcRule.Spec.ExternalServices) {
			return true
		}
	case *kubeovn.ExternalService:
		oldExt := old.(*kubeovn.ExternalService)
		newExt := new.(*kubeovn.ExternalService)
		if oldExt.ResourceVersion == newExt.ResourceVersion {
			return false
		}
		if oldExt.ResourceVersion == newExt.ResourceVersion {
			return true
		}
		if !reflect.DeepEqual(oldExt.Spec.Addresses, newExt.Spec.Addresses) {
			return true
		}
		if !reflect.DeepEqual(oldExt.Spec.Ports, newExt.Spec.Ports) {
			return true
		}
	case *snat.ExternalIPRule:
		oldEipRule := old.(*snat.ExternalIPRule)
		newEipRule := new.(*snat.ExternalIPRule)
		if oldEipRule.ResourceVersion == newEipRule.ResourceVersion {
			return false
		}
		if !reflect.DeepEqual(oldEipRule.Spec, newEipRule.Spec) {
			return true
		}
	case *corev1.Endpoints:
		oldEp := old.(*corev1.Endpoints)
		newEp := new.(*corev1.Endpoints)
		if oldEp.Namespace == "kube-system" {
			return false
		}
		nsConfig := as3.GetTenantConfigForNamespace(oldEp.Namespace)
		if nsConfig == nil {
			klog.V(5).Infof("namespace[%s] not in watch range ", oldEp.Namespace)
			return false
		}
		if oldEp.ResourceVersion == newEp.ResourceVersion {
			return false
		}
		if len(newEp.Subsets) == 0 {
			return false
		}
		if !reflect.DeepEqual(oldEp.Subsets, newEp.Subsets) {
			return true
		}
	}
	return false
}
