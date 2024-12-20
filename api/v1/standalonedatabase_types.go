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
	"github.com/kloudlite/operator/toolkit/plugin"
	"github.com/kloudlite/operator/toolkit/reconciler"
	"github.com/kloudlite/operator/toolkit/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobParams struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Resources    types.Resource      `json:"resources,omitempty"`

	JobName string `json:"jobName,omitempty"`
}

type ResourceRef struct {
	metav1.TypeMeta `json:",inline"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
}

// StandaloneDatabaseSpec defines the desired state of StandaloneDatabase.
type StandaloneDatabaseSpec struct {
	JobParams         JobParams   `json:"jobParams,omitempty"`
	ManagedServiceRef ResourceRef `json:"managedServiceRef"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".spec.targetNamespace",name="target-ns",type=string
//+kubebuilder:printcolumn:JSONPath=".status.lastReconcileTime",name=Seen,type=date
//+kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.checks",name=Checks,type=string
//+kubebuilder:printcolumn:JSONPath=".spec.suspend",name=Suspend,type=boolean
//+kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.resource\\.ready",name=Ready,type=string
//+kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// StandaloneDatabase is the Schema for the standalonedatabases API.
type StandaloneDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StandaloneDatabaseSpec `json:"spec,omitempty"`
	Export plugin.Export          `json:"export,omitempty"`

	Output corev1.SecretReference `json:"output,omitempty"`
	Status reconciler.Status      `json:"status,omitempty"`
}

func (s *StandaloneDatabase) EnsureGVK() {
	if s != nil {
		s.SetGroupVersionKind(GroupVersion.WithKind("StandaloneDatabase"))
	}
}

func (s *StandaloneDatabase) GetStatus() *reconciler.Status {
	return &s.Status
}

func (s *StandaloneDatabase) GetEnsuredLabels() map[string]string {
	return map[string]string{}
}

func (s *StandaloneDatabase) GetEnsuredAnnotations() map[string]string {
	return map[string]string{}
}

// +kubebuilder:object:root=true

// StandaloneDatabaseList contains a list of StandaloneDatabase.
type StandaloneDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StandaloneDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StandaloneDatabase{}, &StandaloneDatabaseList{})
}
