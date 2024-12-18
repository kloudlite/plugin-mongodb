/*
Copyright 2024.

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

package v1

import (
	"github.com/kloudlite/operator/toolkit/reconciler"
	types "github.com/kloudlite/operator/toolkit/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StandaloneServiceSpec defines the desired state of StandaloneService.
type StandaloneServiceSpec struct {
	NodeSelector map[string]string         `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration       `json:"tolerations,omitempty"`
	Resources    types.ResourceWithStorage `json:"resources,omitempty"`
}

// StandaloneServiceStatus defines the observed state of StandaloneService.
type StandaloneServiceStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".spec.targetNamespace",name="target-ns",type=string
//+kubebuilder:printcolumn:JSONPath=".status.lastReconcileTime",name=Seen,type=date
//+kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.checks",name=Checks,type=string
//+kubebuilder:printcolumn:JSONPath=".spec.suspend",name=Suspend,type=boolean
//+kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.resource\\.ready",name=Ready,type=string
//+kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// StandaloneService is the Schema for the standaloneservices API.
type StandaloneService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StandaloneServiceSpec  `json:"spec,omitempty"`
	Output corev1.SecretReference `json:"output,omitempty"`
	Status reconciler.Status      `json:"status,omitempty"`
}

func (s *StandaloneService) EnsureGVK() {
	if s != nil {
		s.SetGroupVersionKind(GroupVersion.WithKind("StandaloneService"))
	}
}

func (s *StandaloneService) GetStatus() *reconciler.Status {
	return &s.Status
}

func (s *StandaloneService) GetEnsuredLabels() map[string]string {
	return map[string]string{}
}

func (s *StandaloneService) GetEnsuredAnnotations() map[string]string {
	return map[string]string{}
}

// +kubebuilder:object:root=true

// StandaloneServiceList contains a list of StandaloneService.
type StandaloneServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StandaloneService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StandaloneService{}, &StandaloneServiceList{})
}
