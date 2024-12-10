package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_ExternalIPRule(obj *ExternalIPRule) {
	if obj.Spec.Priority == 0 {
		obj.Spec.Priority = 1000
	}
}
