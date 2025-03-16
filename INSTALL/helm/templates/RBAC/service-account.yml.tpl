---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "serviceaccount.name" . }}
  namespace: {{.Release.Namespace}}

---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ include "serviceaccount.name" . }}
  name: {{.Release.Namespace}}-{{- $serviceAccount -}}-rb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: "ClusterRole"
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: {{$serviceAccount}}
    namespace: {{.Release.Namespace}}
---
