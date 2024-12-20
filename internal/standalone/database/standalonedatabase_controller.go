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

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kloudlite/operator/toolkit/errors"
	fn "github.com/kloudlite/operator/toolkit/functions"
	job_helper "github.com/kloudlite/operator/toolkit/job-helper"
	"github.com/kloudlite/operator/toolkit/kubectl"
	"github.com/kloudlite/operator/toolkit/reconciler"
	stepResult "github.com/kloudlite/operator/toolkit/reconciler/step-result"
	v1 "github.com/kloudlite/plugin-mongodb/api/v1"
	"github.com/kloudlite/plugin-mongodb/internal/standalone/database/templates"
	"github.com/kloudlite/plugin-mongodb/internal/standalone/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reconciler reconciles a StandaloneDatabase object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Env    *Env

	YAMLClient kubectl.YAMLClient

	templateCreateDBJob []byte
}

// GetName implements reconciler.Reconciler.
func (r *Reconciler) GetName() string {
	return "plugin-mongodb-standalone-database"
}

const (
	createDBCreds  string = "create-db-creds"
	patchDefaults  string = "patch-defaults"
	createDBJob    string = "create-db-job"
	processExports string = "process-exports"
)

const (
	labelJobType       string = "plugin-mongodb.kloudlite.github.com/job.type"
	labelJobGeneration string = "plugin-mongodb.kloudlite.github.com/job.generation"
)

// +kubebuilder:rbac:groups=plugin-mongodb.kloudlite.github.com,resources=standalonedatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=plugin-mongodb.kloudlite.github.com,resources=standalonedatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=plugin-mongodb.kloudlite.github.com,resources=standalonedatabases/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	req, err := reconciler.NewRequest(ctx, r.Client, request.NamespacedName, &v1.StandaloneDatabase{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if req.Object.GetDeletionTimestamp() != nil {
		if x := r.finalize(req); !x.ShouldProceed() {
			return x.ReconcilerResponse()
		}
		return ctrl.Result{}, nil
	}

	req.PreReconcile()
	defer req.PostReconcile()

	if step := req.ClearStatusIfAnnotated(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureCheckList([]reconciler.CheckMeta{
		{Name: patchDefaults, Title: "Patch Defaults"},
		{Name: createDBCreds, Title: "Create DB Credentials"},
		{Name: createDBJob, Title: "Create DB Job"},
		{Name: processExports, Title: "process exports"},
	}); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureLabelsAndAnnotations(); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := req.EnsureFinalizers(reconciler.ForegroundFinalizer, reconciler.CommonFinalizer); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.patchDefaults(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.createDBCreds(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.createDBJob(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	if step := r.processExports(req); !step.ShouldProceed() {
		return step.ReconcilerResponse()
	}

	req.Object.Status.IsReady = true
	return ctrl.Result{}, nil
}

func (r *Reconciler) finalize(req *reconciler.Request[*v1.StandaloneDatabase]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck("finalizing", req)

	ss, err := reconciler.Get(ctx, r.Client, fn.NN(obj.Spec.ManagedServiceRef.Namespace, obj.Spec.ManagedServiceRef.Name), &v1.StandaloneService{})
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return check.Failed(err)
		}
		ss = nil
	}

	if ss == nil || ss.DeletionTimestamp != nil {
		if step := req.ForceCleanupOwnedResources(check); !step.ShouldProceed() {
			return step
		}
	} else {
		if step := req.CleanupOwnedResources(check); !step.ShouldProceed() {
			return step
		}
	}

	if err := fn.DeleteAndWait(ctx, req.Logger, r.Client, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Object.Output.Name,
			Namespace: req.Object.Namespace,
		},
	}); err != nil {
		return check.Failed(err)
	}

	return req.Finalize()
}

func (r *Reconciler) patchDefaults(req *reconciler.Request[*v1.StandaloneDatabase]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(patchDefaults, req)

	hasUpdate := false
	if obj.Output.Name == "" {
		hasUpdate = true
		obj.Output.Name = fmt.Sprintf("output-%s", obj.Name)
	}

	if obj.Spec.JobParams.JobName == "" {
		hasUpdate = true
		obj.Spec.JobParams.JobName = fmt.Sprintf("mongo-database-%s-%s", obj.Name, strings.ToLower(fn.CleanerNanoid(8)))
	}

	if obj.Export.Template == "" {
		hasUpdate = true
		replacer := strings.NewReplacer("$SECRET_NAME$", obj.Output.Name)
		obj.Export.Template = replacer.Replace(strings.TrimSpace(`
USERNAME: {{ secret "$SECRET_NAME$/ROOT_USERNAME" }}
PASSWORD: {{ secret "$SECRET_NAME$/ROOT_PASSWORD" }}
HOST: {{ secret "$SECRET_NAME$/HOST" }}
PORT: {{ secret "$SECRET_NAME$/PORT" }}
DB_NAME: {{ secret "$SECRET_NAME$/DB_NAME" }}
URI: {{ secret "$SECRET_NAME$/URI" }}
`))
	}

	if hasUpdate {
		if err := r.Update(ctx, obj); err != nil {
			return check.Failed(err)
		}

		return check.StillRunning(fmt.Errorf("waiting for resource to re-sync"))
	}

	return check.Completed()
}

func (r *Reconciler) createDBCreds(req *reconciler.Request[*v1.StandaloneDatabase]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(createDBCreds, req)

	msvc, err := reconciler.Get(ctx, r.Client, fn.NN(obj.Spec.ManagedServiceRef.Namespace, obj.Spec.ManagedServiceRef.Name), &v1.StandaloneService{})
	if err != nil {
		return check.Failed(err)
	}

	msvcCreds, err := reconciler.Get(ctx, r.Client, fn.NN(msvc.Namespace, msvc.Output.Name), &corev1.Secret{})
	if err != nil {
		return check.Failed(err)
	}

	req.KV.Set("msvc-output-name", msvc.Output.Name)

	so, err := fn.ParseFromSecretData[types.ServiceCredentials](msvcCreds.Data)
	if err != nil {
		check.Failed(err)
	}

	creds := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: obj.Output.Name, Namespace: obj.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, creds, func() error {
		for k, v := range fn.MapFilterWithPrefix(obj.GetLabels(), "kloudlite.io/") {
			fn.MapSet(&creds.Labels, k, v)
		}
		// creds.SetOwnerReferences([]metav1.OwnerReference{fn.AsOwner(obj, true)})

		if creds.Data == nil {
			username := obj.Name
			password := fn.CleanerNanoid(40)

			dbName := obj.Name

			// INFO: secret does not exist yet
			out := &types.DatabaseOutput{
				Username: username,
				Password: password,
				DbName:   dbName,
				Port:     so.Port,
				Host:     so.Host,
				URI:      fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, so.Host, so.Port, dbName, dbName),

				ClusterLocalHost: so.ClusterLocalHost,
				ClusterLocalURI:  fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, so.ClusterLocalHost, so.Port, dbName, dbName),

				MultiClusterHost: so.MultiClusterHost,
				MultiClusterURI:  fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, so.MultiClusterHost, so.Port, dbName, dbName),

				KloudliteHost: so.KloudliteHost,
				KloudliteURI:  fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=%s", username, password, so.KloudliteHost, so.Port, dbName, dbName),
			}

			m, err := out.ToMap()
			if err != nil {
				return err
			}

			creds.StringData = m
		}
		return nil
	}); err != nil {
		return check.Failed(err)
	}

	return check.Completed()
}

func (r *Reconciler) createDBJob(req *reconciler.Request[*v1.StandaloneDatabase]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(createDBJob, req)

	value, err := req.KV.Get("msvc-output-name")
	if err != nil {
		return check.Failed(err)
	}

	msvcOutputName, ok := value.(string)
	if !ok {
		return check.Failed(fmt.Errorf("msvc-output-name must be a string"))
	}

	job := &batchv1.Job{}
	if err := r.Get(ctx, fn.NN(obj.Namespace, obj.Spec.JobParams.JobName), job); err != nil {
		if !apiErrors.IsNotFound(err) {
			return check.Failed(err)
		}
		job = nil
	}

	// if v, ok := obj.GetAnnotations()[reconciler.ForceReconcile]; ok && v == "true" {
	// 	if helmValues == nil {
	// 		helmValues = make(map[string]apiextensionsv1.JSON, 1)
	// 	}
	// 	b, _ := json.Marshal(map[string]any{"time": time.Now().Format(time.RFC3339)})
	// 	helmValues["force-reconciled-at"] = apiextensionsv1.JSON{Raw: b}
	// 	ann := obj.GetAnnotations()
	// 	delete(ann, constants.ForceReconcile)
	// 	obj.SetAnnotations(ann)
	// 	if err := r.Update(ctx, obj); err != nil {
	// 		return check.StillRunning(fmt.Errorf("waiting for reconcilation"))
	// 	}
	// }

	if job == nil {
		b, err := templates.ParseBytes(r.templateCreateDBJob, templates.CreateDBJobParams{
			JobMetadata: metav1.ObjectMeta{
				Name:      obj.Spec.JobParams.JobName,
				Namespace: obj.Namespace,
				Labels: map[string]string{
					labelJobType:       "create",
					labelJobGeneration: fmt.Sprintf("%d", obj.Generation),
				},
				Annotations:     map[string]string{},
				OwnerReferences: []metav1.OwnerReference{fn.AsOwner(obj, true)},
			},
			PodAnnotations: fn.MapFilterWithPrefix(obj.GetAnnotations(), "kloudlite.io/observability."),
			Tolerations:    obj.Spec.JobParams.Tolerations,
			NodeSelector:   obj.Spec.JobParams.NodeSelector,

			RootUserCredentialsSecret: msvcOutputName,
			NewUserCredentialsSecret:  obj.Output.Name,
		})
		if err != nil {
			return check.Failed(err).NoRequeue()
		}

		rr, err := r.YAMLClient.ApplyYAML(ctx, b)
		if err != nil {
			return check.Failed(err)
		}

		req.AddToOwnedResources(rr...)
		return check.StillRunning(fmt.Errorf("waiting for job to be created")).RequeueAfter(1 * time.Second)
	}

	isMyJob := job.Labels[labelJobGeneration] == fmt.Sprintf("%d", obj.Generation) && job.Labels[labelJobType] == "create"

	if !isMyJob {
		if !job_helper.HasJobFinished(ctx, r.Client, job) {
			return check.Failed(fmt.Errorf("waiting for previous jobs to finish execution"))
		}

		if err := job_helper.DeleteJob(ctx, r.Client, job.Namespace, job.Name); err != nil {
			return check.Failed(err)
		}

		return check.StillRunning(fmt.Errorf("waiting for job to be created")).NoRequeue()
	}

	if !job_helper.HasJobFinished(ctx, r.Client, job) {
		return check.StillRunning(fmt.Errorf("waiting for running job to finish")).NoRequeue()
	}

	check.Message = job_helper.GetTerminationLog(ctx, r.Client, job.Namespace, job.Name)
	if job.Status.Failed > 0 {
		return check.Failed(fmt.Errorf("install or upgrade job failed"))
	}

	return check.Completed()
}

func (r *Reconciler) processExports(req *reconciler.Request[*v1.StandaloneDatabase]) stepResult.Result {
	ctx, obj := req.Context(), req.Object
	check := reconciler.NewRunningCheck(processExports, req)

	if obj.Export.Template == "" {
		return check.Completed()
	}

	if obj.Export.ViaSecret == "" {
		return check.Failed(fmt.Errorf("exports.viaSecret must be specified"))
	}

	valuesMap := struct {
		HelmReleaseName string
	}{
		HelmReleaseName: obj.Name,
	}

	m, err := obj.Export.ParseKV(ctx, r.Client, obj.Namespace, valuesMap)
	if err != nil {
		return check.Failed(errors.NewEf(err, ""))
	}

	exportSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: obj.Export.ViaSecret, Namespace: obj.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, exportSecret, func() error {
		exportSecret.Data = nil
		exportSecret.StringData = m
		return nil
	}); err != nil {
		return check.Failed(errors.NewEf(err, "creating/updating export secret"))
	}

	return check.Completed()
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Client == nil {
		r.Client = mgr.GetClient()
	}

	if r.Scheme == nil {
		r.Scheme = mgr.GetScheme()
	}

	if r.Env == nil {
		return fmt.Errorf("env must be set")
	}

	if r.YAMLClient == nil {
		return fmt.Errorf("yaml client must be set")
	}

	var err error
	r.templateCreateDBJob, err = templates.Read(templates.CreateDBJobTemplate)
	if err != nil {
		return err
	}

	er := mgr.GetEventRecorderFor(r.GetName())

	builder := ctrl.NewControllerManagedBy(mgr).For(&v1.StandaloneDatabase{}).Named(r.GetName())
	builder.Owns(&corev1.Secret{})
	builder.Owns(&batchv1.Job{})
	builder.WithOptions(controller.Options{MaxConcurrentReconciles: r.Env.MaxConcurrentReconciles})
	builder.WithEventFilter(reconciler.ReconcileFilter(er))

	return builder.Complete(r)
}
