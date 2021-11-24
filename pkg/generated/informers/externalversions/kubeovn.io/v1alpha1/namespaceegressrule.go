/*
Copyright 2021 The Kube-OVN CES Controller Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	kubeovniov1alpha1 "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	versioned "github.com/kubeovn/ces-controller/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/kubeovn/ces-controller/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kubeovn/ces-controller/pkg/generated/listers/kubeovn.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// NamespaceEgressRuleInformer provides access to a shared informer and lister for
// NamespaceEgressRules.
type NamespaceEgressRuleInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.NamespaceEgressRuleLister
}

type namespaceEgressRuleInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewNamespaceEgressRuleInformer constructs a new informer for NamespaceEgressRule type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNamespaceEgressRuleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNamespaceEgressRuleInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredNamespaceEgressRuleInformer constructs a new informer for NamespaceEgressRule type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredNamespaceEgressRuleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubeovnV1alpha1().NamespaceEgressRules(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubeovnV1alpha1().NamespaceEgressRules(namespace).Watch(context.TODO(), options)
			},
		},
		&kubeovniov1alpha1.NamespaceEgressRule{},
		resyncPeriod,
		indexers,
	)
}

func (f *namespaceEgressRuleInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNamespaceEgressRuleInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *namespaceEgressRuleInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kubeovniov1alpha1.NamespaceEgressRule{}, f.defaultInformer)
}

func (f *namespaceEgressRuleInformer) Lister() v1alpha1.NamespaceEgressRuleLister {
	return v1alpha1.NewNamespaceEgressRuleLister(f.Informer().GetIndexer())
}