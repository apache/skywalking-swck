// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

// OAPServerConfigReconciler reconciles a OAPServerConfig object
type OAPServerConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type SortByFileName []operatorv1alpha1.FileConfig

func (a SortByFileName) Len() int {
	return len(a)
}
func (a SortByFileName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a SortByFileName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

type SortByEnvName []core.EnvVar

func (a SortByEnvName) Len() int {
	return len(a)
}
func (a SortByEnvName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a SortByEnvName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *OAPServerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================oapserverconfig reconcile started================================")

	oapServerConfig := operatorv1alpha1.OAPServerConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, &oapServerConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	oapList := operatorv1alpha1.OAPServerList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	if err := r.List(ctx, &oapList, opts...); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to list oapserver: %w", err)
	}

	// get the specific version's oapserver
	for i := range oapList.Items {
		if oapList.Items[i].Spec.Version == oapServerConfig.Spec.Version {
			oapServer := oapList.Items[i]
			deployment := apps.Deployment{}
			if err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: oapServer.Name + "-oap"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("failed to get the deployment of OAPServer: %w", err)
			}
			// overlay the env configuration
			envChanged, err := r.OverlayEnv(log, &oapServerConfig, &deployment)
			if err != nil {
				log.Error(err, "failed to overlay the env configuration")
			}
			// overlay the file configuration
			fileChanged, err := r.OverlayStaticFile(ctx, log, &oapServerConfig, &deployment)
			if err != nil {
				log.Error(err, "failed to overlay the file configuration")
			}
			// update the deployment
			if envChanged || fileChanged {
				if err := r.Client.Update(ctx, &deployment); err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to update the deployment of OAPServer: %w", err)
				}
			}
		}
	}

	if err := r.checkState(ctx, log, &oapServerConfig, oapList); err != nil {
		log.Error(err, "failed to update OAPServerConfig's status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *OAPServerConfigReconciler) OverlayEnv(log logr.Logger,
	oapServerConfig *operatorv1alpha1.OAPServerConfig, deployment *apps.Deployment) (bool, error) {
	changed := false

	sort.Sort(SortByEnvName(oapServerConfig.Spec.Env))
	newMd5Hash := MD5Hash(oapServerConfig.Spec.Env)

	// Overlay, not replace. The OAPServer controller puts the storage wiring into this container's
	// environment -- SW_STORAGE and, for BanyanDB, SW_STORAGE_BANYANDB_TARGETS -- along with
	// JAVA_OPTS. Assigning Spec.Env over the top dropped all of it, and the OAP that came back
	// looked for a database on 127.0.0.1.
	//
	// That went unnoticed for as long as it did because SkyWalking once had an embedded H2: an OAP
	// that lost SW_STORAGE fell back to it and still worked. H2 was removed in 10.2.0 and the
	// selector defaults to banyandb, so with no fallback left the same silent loss now leaves the
	// OAP dialling localhost forever.
	//
	// Overlaying alone is not enough either: a name dropped from spec.env has to come back off the
	// container, and if the overlay had shadowed a template value -- JAVA_OPTS is -Xmx2048M in the
	// OAPServer template -- that value has to be restored rather than deleted. What the overlay
	// replaced is therefore recorded alongside it, because by the time the next reconcile reads the
	// live deployment the original is already gone.
	previous, err := readEnvOverlay(deployment)
	if err != nil {
		return false, err
	}
	merged, originals := applyEnvOverlay(
		deployment.Spec.Template.Spec.Containers[0].Env, oapServerConfig.Spec.Env, previous)

	oldMd5Hash, ok := deployment.Spec.Template.Labels["md5-env"]
	if !ok || oldMd5Hash != newMd5Hash {
		changed = true
	}
	// Also restore an overlay that something else has since flattened, which the hash alone --
	// computed only over the OAPServerConfig's own entries -- cannot see.
	if !apiequal.Semantic.DeepEqual(deployment.Spec.Template.Spec.Containers[0].Env, merged) {
		changed = true
	}

	if changed {
		deployment.Spec.Template.Spec.Containers[0].Env = merged
		deployment.Spec.Template.Labels["md5-env"] = newMd5Hash
		if err := writeEnvOverlay(deployment, originals); err != nil {
			return false, err
		}
	} else {
		log.Info("env configuration keeps the same as before")
		return changed, nil
	}

	log.Info("successfully overlay the env configuration")
	return changed, nil
}

// overlayEnv returns base with overrides applied by name: an override replaces the entry of the
// same name in place, and anything new is appended. Order is otherwise base's, so a deployment
// does not churn just because the overlay was re-evaluated.
func overlayEnv(base, overrides []core.EnvVar) []core.EnvVar {
	byName := make(map[string]core.EnvVar, len(overrides))
	for _, e := range overrides {
		byName[e.Name] = e
	}

	merged := make([]core.EnvVar, 0, len(base)+len(overrides))
	seen := make(map[string]bool, len(base))
	for _, e := range base {
		if override, ok := byName[e.Name]; ok {
			merged = append(merged, override)
		} else {
			merged = append(merged, e)
		}
		seen[e.Name] = true
	}
	for _, e := range overrides {
		if !seen[e.Name] {
			merged = append(merged, e)
		}
	}
	return merged
}

// envOverlayAnnotation records the names an OAPServerConfig overlaid onto the OAP container and
// what each held beforehand. Without it a name removed from spec.env would linger on the
// deployment forever: the next reconcile reads the live container, where the overlaid value is
// indistinguishable from one the OAPServer template set itself.
const envOverlayAnnotation = "operator.skywalking.apache.org/env-overlay"

// shadowedEnv is what a name held before the overlay first took it over. Existed=false means the
// overlay introduced the name, so dropping it removes the variable outright.
type shadowedEnv struct {
	Existed bool        `json:"existed"`
	Value   core.EnvVar `json:"value,omitempty"`
}

func readEnvOverlay(deployment *apps.Deployment) (map[string]shadowedEnv, error) {
	raw, ok := deployment.Spec.Template.Annotations[envOverlayAnnotation]
	if !ok || raw == "" {
		return map[string]shadowedEnv{}, nil
	}
	shadowed := map[string]shadowedEnv{}
	if err := json.Unmarshal([]byte(raw), &shadowed); err != nil {
		// A hand-edited or truncated annotation must not wedge the controller: start over rather
		// than refusing to reconcile, accepting that one generation of originals is lost.
		return map[string]shadowedEnv{}, nil
	}
	return shadowed, nil
}

func writeEnvOverlay(deployment *apps.Deployment, shadowed map[string]shadowedEnv) error {
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	if len(shadowed) == 0 {
		delete(deployment.Spec.Template.Annotations, envOverlayAnnotation)
		return nil
	}
	raw, err := json.Marshal(shadowed)
	if err != nil {
		return err
	}
	deployment.Spec.Template.Annotations[envOverlayAnnotation] = string(raw)
	return nil
}

// applyEnvOverlay puts back whatever the previous overlay displaced for names the OAPServerConfig
// no longer sets, then applies the current ones. It returns the merged environment and the
// originals to record for next time.
func applyEnvOverlay(base, overrides []core.EnvVar,
	previous map[string]shadowedEnv) ([]core.EnvVar, map[string]shadowedEnv) {
	wanted := make(map[string]bool, len(overrides))
	for _, e := range overrides {
		wanted[e.Name] = true
	}

	// Undo overlays that are no longer requested.
	restored := make([]core.EnvVar, 0, len(base)+len(overrides))
	for _, e := range base {
		if wanted[e.Name] {
			restored = append(restored, e) // overlayEnv replaces it below
			continue
		}
		if orig, overlaid := previous[e.Name]; overlaid {
			if !orig.Existed {
				continue // the overlay introduced this name; drop it with the overlay
			}
			restored = append(restored, orig.Value)
			continue
		}
		restored = append(restored, e)
	}

	// Record what the current overlay shadows, keeping the first original ever seen for a name so
	// repeated edits cannot ratchet the "original" forward into an overlaid value.
	current := make(map[string]core.EnvVar, len(restored))
	for _, e := range restored {
		current[e.Name] = e
	}
	originals := make(map[string]shadowedEnv, len(overrides))
	for _, e := range overrides {
		if orig, ok := previous[e.Name]; ok {
			originals[e.Name] = orig
		} else if cur, ok := current[e.Name]; ok {
			originals[e.Name] = shadowedEnv{Existed: true, Value: cur}
		} else {
			originals[e.Name] = shadowedEnv{Existed: false}
		}
	}

	return overlayEnv(restored, overrides), originals
}

// mergeVolumeMounts returns base with additions applied by name: an addition replaces the mount of
// the same name in place, and anything new is appended. Order is otherwise base's, so re-running
// the overlay does not churn the pod template.
func mergeVolumeMounts(base, additions []core.VolumeMount) []core.VolumeMount {
	byName := make(map[string]core.VolumeMount, len(additions))
	for _, m := range additions {
		byName[m.Name] = m
	}
	merged := make([]core.VolumeMount, 0, len(base)+len(additions))
	seen := make(map[string]bool, len(base))
	for _, m := range base {
		if replacement, ok := byName[m.Name]; ok {
			merged = append(merged, replacement)
		} else {
			merged = append(merged, m)
		}
		seen[m.Name] = true
	}
	for _, m := range additions {
		if !seen[m.Name] {
			merged = append(merged, m)
		}
	}
	return merged
}

// mergeVolumes is mergeVolumeMounts for volumes.
func mergeVolumes(base, additions []core.Volume) []core.Volume {
	byName := make(map[string]core.Volume, len(additions))
	for _, v := range additions {
		byName[v.Name] = v
	}
	merged := make([]core.Volume, 0, len(base)+len(additions))
	seen := make(map[string]bool, len(base))
	for _, v := range base {
		if replacement, ok := byName[v.Name]; ok {
			merged = append(merged, replacement)
		} else {
			merged = append(merged, v)
		}
		seen[v.Name] = true
	}
	for _, v := range additions {
		if !seen[v.Name] {
			merged = append(merged, v)
		}
	}
	return merged
}

func (r *OAPServerConfigReconciler) OverlayStaticFile(ctx context.Context, log logr.Logger,
	oapServerConfig *operatorv1alpha1.OAPServerConfig, deployment *apps.Deployment) (bool, error) {
	changed := false

	sort.Sort(SortByFileName(oapServerConfig.Spec.File))
	newMd5Hash := MD5Hash(oapServerConfig.Spec.File)
	configmap := core.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServerConfig.Namespace,
		Name: oapServerConfig.Name}, &configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get the static file configuration's configmap")
		return changed, err
	}
	// Whether the ConfigMap needs rebuilding is a separate question from whether the pod still
	// mounts it. Returning early on an unchanged hash meant that once the OAPServer controller
	// re-rendered the Deployment -- which it does on any spec change, storage change or operator
	// upgrade, and which drops volumes it does not know about -- the static file mount was never
	// put back. So only the ConfigMap rebuild is gated on the hash; the mount is reconciled every
	// pass, and `changed` is set from whether the pod template actually differs.
	configMapExists := !apierrors.IsNotFound(err)
	rebuildConfigMap := !configMapExists || configmap.Labels["md5-file"] != newMd5Hash
	if configMapExists && rebuildConfigMap {
		if err := r.Client.Delete(ctx, &configmap); err != nil {
			log.Error(err, "faled to delete the static file configuration's configmap")
		}
	}

	data := make(map[string]string)
	mounts := []core.VolumeMount{}
	volume := core.Volume{
		Name: oapServerConfig.Name,
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: oapServerConfig.Name,
				},
			},
		},
	}
	for _, f := range oapServerConfig.Spec.File {
		mounts = append(mounts, core.VolumeMount{
			MountPath: f.Path + "/" + f.Name,
			Name:      oapServerConfig.Name,
			SubPath:   f.Name,
		})
		data[f.Name] = f.Data
	}

	labels := make(map[string]string)
	// set the version label
	labels["version"] = oapServerConfig.Spec.Version
	// set the configuration type
	labels["oapServerConfig"] = "static"
	// set the md5 value of the data
	labels["md5-file"] = newMd5Hash
	// create configmap for static files
	configmap = core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oapServerConfig.Name,
			Namespace: oapServerConfig.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: oapServerConfig.APIVersion,
					Kind:       oapServerConfig.Kind,
					Name:       oapServerConfig.Name,
					UID:        oapServerConfig.UID,
				},
			},
			Labels: labels,
		},
		Data: data,
	}

	if rebuildConfigMap {
		changed = true
		if err := r.Client.Create(ctx, &configmap); err != nil {
			log.Error(err, "failed to create static configuration configmap")
			return changed, err
		}
	}

	// Merge, do not assign. ApplyOverlay is an RFC 7386 merge patch, under which an array in the
	// overlay REPLACES the array it patches -- so assigning here dropped whatever else the pod
	// carried, which for a TLS storage is the volume holding its certificate, leaving the OAP with
	// SW_STORAGE_BANYANDB_SSL_TRUST_CA_PATH pointing at nothing.
	mergedMounts := mergeVolumeMounts(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, mounts)
	mergedVolumes := mergeVolumes(deployment.Spec.Template.Spec.Volumes, []core.Volume{volume})
	if !apiequal.Semantic.DeepEqual(deployment.Spec.Template.Spec.Containers[0].VolumeMounts, mergedMounts) ||
		!apiequal.Semantic.DeepEqual(deployment.Spec.Template.Spec.Volumes, mergedVolumes) {
		changed = true
	}
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = mergedMounts
	deployment.Spec.Template.Spec.Volumes = mergedVolumes

	log.Info("successfully overlay the file configuration")
	return changed, nil
}

func (r *OAPServerConfigReconciler) checkState(ctx context.Context, log logr.Logger,
	oapServerConfig *operatorv1alpha1.OAPServerConfig, oapList operatorv1alpha1.OAPServerList) error {
	errCol := new(kubernetes.ErrorCollector)

	nilTime := metav1.Time{}
	now := metav1.NewTime(time.Now())
	overlay := operatorv1alpha1.OAPServerConfigStatus{}

	// get Instances and AvailableReplicas
	for i := range oapList.Items {
		if oapList.Items[i].Spec.Version == oapServerConfig.Spec.Version {
			overlay.Desired += int(oapList.Items[i].Spec.Instances)
			overlay.Ready += int(oapList.Items[i].Status.AvailableReplicas)
		}
	}

	if oapServerConfig.Status.CreationTime == nilTime {
		overlay.CreationTime = now
		overlay.LastUpdateTime = now
	} else {
		overlay.CreationTime = oapServerConfig.Status.CreationTime
		overlay.LastUpdateTime = now
	}

	oapServerConfig.Status = overlay
	oapServerConfig.Kind = "OAPServerConfig"
	if err := kubernetes.ApplyOverlay(oapServerConfig, &operatorv1alpha1.OAPServerConfig{Status: overlay}); err != nil {
		errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		return errCol.Error()
	}

	if err := r.updateStatus(ctx, oapServerConfig, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of OAPServerConfig: %w", err))
	}

	log.Info("updated OAPServerConfig sub resource")
	return errCol.Error()
}

func (r *OAPServerConfigReconciler) updateStatus(ctx context.Context, oapServerConfig *operatorv1alpha1.OAPServerConfig,
	overlay operatorv1alpha1.OAPServerConfigStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: oapServerConfig.Name, Namespace: oapServerConfig.Namespace}, oapServerConfig); err != nil {
			errCol.Collect(fmt.Errorf("failed to get oapServerConfig: %w", err))
		}
		oapServerConfig.Status = overlay
		oapServerConfig.Kind = "OAPServerConfig"
		if err := kubernetes.ApplyOverlay(oapServerConfig, &operatorv1alpha1.OAPServerConfig{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, oapServerConfig); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status: %w", err))
		}
		return errCol.Error()
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *OAPServerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.OAPServerConfig{}).
		Owns(&apps.Deployment{}).
		Complete(r)
}
