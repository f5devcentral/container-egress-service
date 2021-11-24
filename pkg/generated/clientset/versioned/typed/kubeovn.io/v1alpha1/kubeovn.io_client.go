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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kubeovn/ces-controller/pkg/apis/kubeovn.io/v1alpha1"
	"github.com/kubeovn/ces-controller/pkg/generated/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type KubeovnV1alpha1Interface interface {
	RESTClient() rest.Interface
	ClusterEgressRulesGetter
	ExternalServicesGetter
	NamespaceEgressRulesGetter
	ServiceEgressRulesGetter
}

// KubeovnV1alpha1Client is used to interact with features provided by the kubeovn.io/v1alpha1 group.
type KubeovnV1alpha1Client struct {
	restClient rest.Interface
}

func (c *KubeovnV1alpha1Client) ClusterEgressRules() ClusterEgressRuleInterface {
	return newClusterEgressRules(c)
}

func (c *KubeovnV1alpha1Client) ExternalServices(namespace string) ExternalServiceInterface {
	return newExternalServices(c, namespace)
}

func (c *KubeovnV1alpha1Client) NamespaceEgressRules(namespace string) NamespaceEgressRuleInterface {
	return newNamespaceEgressRules(c, namespace)
}

func (c *KubeovnV1alpha1Client) ServiceEgressRules(namespace string) ServiceEgressRuleInterface {
	return newServiceEgressRules(c, namespace)
}

// NewForConfig creates a new KubeovnV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*KubeovnV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &KubeovnV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new KubeovnV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *KubeovnV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new KubeovnV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *KubeovnV1alpha1Client {
	return &KubeovnV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *KubeovnV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}