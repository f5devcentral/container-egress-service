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

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kubeovn/ces-controller/pkg/as3"
	clientset "github.com/kubeovn/ces-controller/pkg/generated/clientset/versioned"
	informers "github.com/kubeovn/ces-controller/pkg/generated/informers/externalversions"
	"github.com/kubeovn/ces-controller/pkg/signals"

	"github.com/kubeovn/ces-controller/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	masterURL  string
	kubeconfig string

	bigipURL      string
	bigipInsecure bool
	bigipUsername string
	bigipPassword string
	bigipCredsDir string

	gateway string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	if bigipCredsDir != "" {
		usernameFile := filepath.Join(bigipCredsDir, "username")
		passwordFile := filepath.Join(bigipCredsDir, "password")

		setField := func(field *string, filename, fieldType string) error {
			fileBytes, readErr := ioutil.ReadFile(filename)
			if readErr != nil {
				klog.Infof("No %s in credentials directory, falling back to command argument", fieldType)
				if len(*field) == 0 {
					return fmt.Errorf(fmt.Sprintf("Big-IP %s not specified", fieldType))
				}
			} else {
				*field = string(fileBytes)
			}

			return nil
		}

		if err := setField(&bigipUsername, usernameFile, "username"); err != nil {
			panic(err)
		}
		if err := setField(&bigipPassword, passwordFile, "password"); err != nil {
			panic(err)
		}
	}

	if bigipUsername == "" || bigipPassword == "" {
		panic("Missing Big-IP credentials info")
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		ns = []byte("default")
		//klog.Fatalf("Error reading /var/run/secrets/kubernetes.io/serviceaccount/namespace: %v", err)
	}

	
	cm, err := kubeClient.CoreV1().ConfigMaps(string(bytes.TrimSpace(ns))).Get(context.Background(), controller.ControllerConfigmap, metav1.GetOptions{})
	if err != nil {
		klog.Fatalf("failed to get configmap[%s]: %v", controller.ControllerConfigmap, err)
	}

	var initialized bool
	if s := cm.Data["initialized"]; s != "" {
		if initialized, err = strconv.ParseBool(s); err != nil {
			klog.Fatalf("failed to parse bool value in configmap[%s]: %v", controller.ControllerConfigmap, err)
		}
	}

	err = as3.InitAs3Tenant(adcInitTemplate, as3.NewClient(bigipURL, bigipUsername, bigipPassword, bigipInsecure), initialized)
	if err != nil {
		klog.Fatalf("failed to initialize AS3 declaration: %v", err)
	}
	if !initialized {
		cm.Data["initialized"] = "true"
		if _, err = kubeClient.CoreV1().ConfigMaps(string(bytes.TrimSpace(ns))).Update(context.Background(), cm, metav1.UpdateOptions{}); err != nil {
			klog.Fatalf("failed to update configmap[%s]: %v", controller.ControllerConfigmap, err)
		}
	}

	as3Client, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building AS3 clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	as3InformerFactory := informers.NewSharedInformerFactory(as3Client, time.Second*30)
	endpointsInformer := kubeInformerFactory.Core().V1().Endpoints()
	externalServiceInformer := as3InformerFactory.Kubeovn().V1alpha1().ExternalServices()
	clusterEgressRuleInformer := as3InformerFactory.Kubeovn().V1alpha1().ClusterEgressRules()
	namespaceEgressRuleInformer := as3InformerFactory.Kubeovn().V1alpha1().NamespaceEgressRules()
	serviceEgressRuleInformer := as3InformerFactory.Kubeovn().V1alpha1().ServiceEgressRules()

	controller := controller.NewController(kubeClient, as3Client, endpointsInformer, externalServiceInformer, clusterEgressRuleInformer,
		namespaceEgressRuleInformer, serviceEgressRuleInformer, as3.NewClient(bigipURL, bigipUsername, bigipPassword, bigipInsecure))

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	as3InformerFactory.Start(stopCh)

	if err = controller.Run(stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	flag.StringVar(&bigipURL, "bigip-url", "", "Required, URL for the Big-IP.")
	flag.BoolVar(&bigipInsecure, "bigip-insecure", false, "Optional, when set to true, enable insecure SSL communication to BigIP.")
	flag.StringVar(&bigipUsername, "bigip-username", "", "User name for the Big-IP user account.")
	flag.StringVar(&bigipPassword, "bigip-password", "", "Password for the Big-IP user account.")
	flag.StringVar(&bigipCredsDir, "bigip-creds-dir", "", "Directory that contains the BIG-IP username and password. To be used instead of username and password.")

	flag.StringVar(&gateway, "gateway", "", "Gateway for egress traffic.")
}
