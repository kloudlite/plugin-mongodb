package templates

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CreateDBJobParams struct {
	JobMetadata metav1.ObjectMeta

	PodLabels      map[string]string
	PodAnnotations map[string]string

	NodeSelector map[string]string
	Tolerations  []corev1.Toleration

	RootUserCredentialsSecret string
	NewUserCredentialsSecret  string
}
