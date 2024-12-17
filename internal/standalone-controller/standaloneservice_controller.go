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

package standalone_controller

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	fn "github.com/kloudlite/operator/toolkit/functions"
	"github.com/kloudlite/operator/toolkit/reconciler"
	stepResult "github.com/kloudlite/operator/toolkit/reconciler/step-result"
	v1 "github.com/kloudlite/plugin-mongodb/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StandaloneServiceReconciler reconciles a StandaloneService object
type StandaloneServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Env    Env
}

// GetName implements reconciler.Reconciler.
func (r *StandaloneServiceReconciler) GetName() string {
	return "plugin-mongodb-standalone-service"
}

const (
	patchDefaults           string = "patch-defaults"
	createPVC               string = "create-pvc"
	createService           string = "create-service"
	createAccessCredentials string = "create-access-credentials"
	createStatefulSet       string = "create-statefulset"

	cleanupOwnedResources string = "cleanupOwnedResources"
)

// +kubebuilder:rbac:groups=plugin-mongodb.kloudlite.github.com,resources=standaloneservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=plugin-mongodb.kloudlite.github.com,resources=standaloneservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=plugin-mongodb.kloudlite.github.com,resources=standaloneservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaim,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the StandaloneService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *StandaloneServiceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := reconciler.NewRequest(ctx, r.Client, request.NamespacedName, &v1.StandaloneService{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	req.PreReconcile()
	defer req.PostReconcile()

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(reconciler.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureCheckList([]reconciler.CheckMeta{
		{Name: patchDefaults, Title: "Defaults Patched", Debug: true},
		{Name: createService, Title: "Access Credentials Generated"},
		{Name: createPVC, Title: "MongoDB Helm Applied"},
		{Name: createAccessCredentials, Title: "MongoDB Helm Ready"},
		{Name: createStatefulSet, Title: "MongoDB StatefulSets Ready"},
	}); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.patchDefaults(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.createService(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.createPVC(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.createAccessCredentials(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.createStatefulSet(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	return ctrl.Result{}, nil
}

func (r *StandaloneServiceReconciler) finalize(req *reconciler.Request[*v1.StandaloneService]) stepResult.Result {
	check := reconciler.NewRunningCheck("finalizing", req)

	if step := req.EnsureCheckList([]reconciler.CheckMeta{
		{Name: "finalizing", Title: "Cleanup Owned Resources"},
	}); !step.ShouldProceed() {
		return step
	}

	if result := req.CleanupOwnedResources(check); !result.ShouldProceed() {
		return result
	}

	return req.Finalize()
}

func (r *StandaloneServiceReconciler) patchDefaults(req *reconciler.Request[*v1.StandaloneService]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck("finalizing", req)

	hasUpdate := false

	if obj.Output.Name == "" {
		hasUpdate = true
		obj.Output.Name = fmt.Sprintf("output-%s", obj.Name)
	}

	if obj.Output.Namespace == "" {
		hasUpdate = true
		obj.Output.Namespace = obj.Namespace
	}

	if hasUpdate {
		if err := r.Update(ctx, obj); err != nil {
			return check.Failed(err)
		}

		return check.StillRunning(fmt.Errorf("waiting for reconcilation"))
	}

	return check.Completed()
}

func getKloudliteDNSHostname(obj *v1.StandaloneService) string {
	return fmt.Sprintf("%s.svc", obj.Name)
}

const (
	StandaloneServiceComponentLabel = "plugin-mongodb.kloudlite.github.com/component"
)

func (r *StandaloneServiceReconciler) createService(req *reconciler.Request[*v1.StandaloneService]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(createService, req)

	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: obj.Name, Namespace: obj.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		lb := svc.GetLabels()
		fn.MapSet(&lb, reconciler.KloudliteDNSHostnameKey, getKloudliteDNSHostname(obj))
		for k, v := range obj.GetLabels() {
			lb[k] = v
		}

		svc.SetLabels(lb)
		svc.SetOwnerReferences([]metav1.OwnerReference{fn.AsOwner(obj, true)})
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:     "mongo",
				Protocol: corev1.ProtocolTCP,
				Port:     27017,
			},
		}

		svc.Spec.Selector = fn.MapMerge(
			fn.MapFilter(obj.GetLabels(), func(k, v string) bool { return !strings.HasPrefix(k, "kloudlite.io/operator.") }),
			map[string]string{StandaloneServiceComponentLabel: "statefulset"},
		)
		return nil
	}); err != nil {
		return check.Failed(err)
	}

	return check.Completed()
}

func (r *StandaloneServiceReconciler) createPVC(req *reconciler.Request[*v1.StandaloneService]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(createPVC, req)

	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: obj.Name, Namespace: obj.Namespace}}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		pvc.SetLabels(obj.GetLabels())
		pvc.SetOwnerReferences([]metav1.OwnerReference{fn.AsOwner(obj, true)})
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}

		if pvc.Spec.Resources.Requests == nil {
			pvc.Spec.Resources.Requests = corev1.ResourceList{}
		}

		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(string(obj.Spec.Resources.Storage.Size))
		if obj.Spec.Resources.Storage.StorageClass != "" {
			pvc.Spec.StorageClassName = &obj.Spec.Resources.Storage.StorageClass
		}
		return nil
	}); err != nil {
		return check.Failed(err)
	}

	return check.Completed()
}

func (r *StandaloneServiceReconciler) createAccessCredentials(req *reconciler.Request[*v1.StandaloneService]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(createAccessCredentials, req)

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: obj.Output.Name, Namespace: obj.Output.Namespace}}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.SetLabels(obj.GetLabels())
		secret.SetOwnerReferences([]metav1.OwnerReference{fn.AsOwner(obj, true)})

		if secret.Data == nil {
			username := "root"
			password := fn.CleanerNanoid(40)

			clusterLocalHost := fmt.Sprintf("%s.%s.svc.%s", obj.Name, obj.Namespace, r.Env.ClusterInternalDNS)
			multiClusterHost := fmt.Sprintf("%s.%s.svc.%s", obj.Name, obj.Namespace, r.Env.MultiClusterDNS)
			kloudliteHost := fmt.Sprintf("%s.%s", getKloudliteDNSHostname(obj), r.Env.KloudliteDNS)
			port := "27017"

			dbName := "admin"
			out := OutputCredentials{
				RootUsername: username,
				RootPassword: password,
				DBName:       dbName,
				AuthSource:   dbName,

				Port: port,

				ClusterLocalHost: clusterLocalHost,
				ClusterLocalAddr: fmt.Sprintf("%s:%s", clusterLocalHost, port),
				ClusterLocalURI:  fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, clusterLocalHost, port, dbName, dbName),

				MultiClusterHost: multiClusterHost,
				MultiClusterAddr: fmt.Sprintf("%s:%s", multiClusterHost, port),
				MultiClusterURI:  fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, multiClusterHost, port, dbName, dbName),

				KloudliteHost: kloudliteHost,
				KloudliteAddr: fmt.Sprintf("%s:%s", kloudliteHost, port),
				KloudliteURI:  fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, kloudliteHost, port, dbName, dbName),

				Host: kloudliteHost,
				Addr: fmt.Sprintf("%s:%s", kloudliteHost, port),
				URI:  fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, kloudliteHost, port, dbName, dbName),
			}

			m, err := out.ToMap()
			if err != nil {
				return err
			}

			secret.StringData = m
		}

		return nil
	}); err != nil {
		return check.Failed(err)
	}

	return check.Completed()
}

func (r *StandaloneServiceReconciler) createStatefulSet(req *reconciler.Request[*v1.StandaloneService]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(createStatefulSet, req)

	pvcName := obj.Name

	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: obj.Name, Namespace: obj.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.SetOwnerReferences([]metav1.OwnerReference{fn.AsOwner(obj, true)})

		selectorLabels := fn.MapMerge(
			fn.MapFilterWithPrefix(obj.GetLabels(), "kloudlite.io/"),
			map[string]string{StandaloneServiceComponentLabel: "statefulset"},
		)

		for k, v := range selectorLabels {
			fn.MapSet(&sts.Labels, k, v)
		}

		spec := appsv1.StatefulSetSpec{
			Replicas: fn.New(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			ServiceName: obj.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: corev1.PodSpec{
					NodeSelector: obj.Spec.NodeSelector,
					Tolerations:  obj.Spec.Tolerations,
					Volumes: []corev1.Volume{
						{
							Name: pvcName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: obj.Name,
									ReadOnly:  false,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "mongodb",
							Image: "mongo:latest",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      pvcName,
									MountPath: "/data/db",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "MONGO_INITDB_ROOT_USERNAME",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: obj.Output.Name,
											},
											Key: "ROOT_USERNAME",
										},
									},
								},
								{
									Name: "MONGO_INITDB_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: obj.Output.Name,
											},
											Key: "ROOT_PASSWORD",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if sts.GetGeneration() > 0 {
			// resource exists, and is being updated now
			// INFO: k8s statefulsets forbids update to spec fields, other than "replicas", "template", "ordinals", "updateStrategy", "persistentVolumeClaimRetentionPolicy" and "minReadySeconds"
			sts.Spec.Replicas = spec.Replicas
			sts.Spec.Template = spec.Template
		} else {
			sts.Spec = spec
		}

		return nil
	}); err != nil {
		return check.Failed(err)
	}

	if sts.Status.Replicas > 0 && sts.Status.ReadyReplicas != sts.Status.Replicas {
		return check.StillRunning(fmt.Errorf("waiting for statefulset pods to start"))
	}

	return check.Completed()
}

// SetupWithManager sets up the controller with the Manager.
func (r *StandaloneServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).For(&v1.StandaloneService{}).Named(r.GetName())
	builder.Owns(&appsv1.StatefulSet{})
	builder.Owns(&corev1.Service{})
	builder.Owns(&corev1.PersistentVolumeClaim{})
	builder.Owns(&corev1.Secret{})
	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.WithEventFilter(reconciler.ReconcileFilter())

	return builder.Complete(r)
}
