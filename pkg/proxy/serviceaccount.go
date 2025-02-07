/*
Copyright 2021 Yaacov Zamir.

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

package proxy

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocgatev1beta1 "github.com/rh-fieldwork/kube-gateway-operator/api/v1beta1"
)

// ServiceAccount is
func ServiceAccount(s *ocgatev1beta1.GateServer) (*corev1.ServiceAccount, error) {
	labels := map[string]string{
		"app": s.Name,
	}

	serviceaccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
			Labels:    labels,
		},
		Secrets: []corev1.ObjectReference{
			{
				Name: fmt.Sprintf("%s-jwt-secret", s.Name),
			},
		},
	}

	return serviceaccount, nil
}
