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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	certv1beta1 "k8s.io/api/certificates/v1beta1"
	core "k8s.io/api/core/v1"
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	pkcs12 "software.sslmate.com/src/go-pkcs12"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

// StorageReconciler reconciles a Storage object
type StorageReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	FileRepo   kubernetes.Repo
	Recorder   record.EventRecorder
	RestConfig *rest.Config
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=storages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;serviceaccounts;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests/approval,verbs=update

func (r *StorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================storage reconcile started================================")

	storage := operatorv1alpha1.Storage{}
	if err := r.Client.Get(ctx, req.NamespacedName, &storage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if storage.Spec.ConnectType == "external" {
		return ctrl.Result{RequeueAfter: schedDuration}, nil
	}

	r.createCert(ctx, log, &storage)
	r.checkSecurity(ctx, log, &storage)

	ff, err := r.FileRepo.GetFilesRecursive(storage.Spec.Type + "/templates")
	if err != nil {
		log.Error(err, "failed to load resource templates")
		return ctrl.Result{}, err
	}
	app := kubernetes.Application{
		Client:   r.Client,
		FileRepo: r.FileRepo,
		CR:       &storage,
		GVK:      operatorv1alpha1.GroupVersion.WithKind("Storage"),
		Recorder: r.Recorder,
		TmplFunc: tmplFunc(),
	}
	if err := app.ApplyAll(ctx, ff, log); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.checkState(ctx, log, &storage); err != nil {
		log.Error(err, "failed to check sub resources state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func tmplFunc() map[string]interface{} {
	return map[string]interface{}{"getProtocol": getProtocol}
}

func getProtocol(tls bool) string {
	if tls {
		return "https"
	}
	return "http"
}

func (r *StorageReconciler) checkState(ctx context.Context, log logr.Logger, storage *operatorv1alpha1.Storage) error {
	overlay := operatorv1alpha1.StorageStatus{}
	statefulset := apps.StatefulSet{}
	errCol := new(kubernetes.ErrorCollector)
	object := client.ObjectKey{Namespace: storage.Namespace, Name: storage.Name + "-" + storage.Spec.Type}
	if err := r.Client.Get(ctx, object, &statefulset); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get statefulset: %w", err))
	} else {
		if statefulset.Status.ReadyReplicas == statefulset.Status.Replicas {
			overlay.Conditions = append(overlay.Conditions, apps.StatefulSetCondition{
				Type:               "Ready",
				Status:             "True",
				LastTransitionTime: metav1.NewTime(time.Now()),
				Reason:             "statefulset " + object.Name + " is ready",
			})
		}
	}

	if apiequal.Semantic.DeepDerivative(overlay, storage.Status) {
		log.Info("Status keeps the same as before")
	}
	storage.Status = overlay
	storage.Kind = "Storage"
	if err := kubernetes.ApplyOverlay(storage, &operatorv1alpha1.Storage{Status: overlay}); err != nil {
		errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		return errCol.Error()
	}
	if err := r.updateStatus(ctx, storage, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of es: %w", err))
	}
	log.Info("updated Status sub resource")
	return errCol.Error()
}

func (r *StorageReconciler) updateStatus(ctx context.Context, storage *operatorv1alpha1.Storage,
	overlay operatorv1alpha1.StorageStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: storage.Name, Namespace: storage.Namespace}, storage); err != nil {
			errCol.Collect(fmt.Errorf("failed to get storage: %w", err))
		}
		storage.Status = overlay
		storage.Kind = "Storage"
		if err := kubernetes.ApplyOverlay(storage, &operatorv1alpha1.Storage{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, storage); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status of storage: %w", err))
		}
		return errCol.Error()
	})
}

func (r *StorageReconciler) checkSecurity(ctx context.Context, log logr.Logger, s *operatorv1alpha1.Storage) {
	user, tls := s.Spec.Security.User, s.Spec.Security.TLS
	if user.SecretName != "" {
		if user.SecretName == "default" {
			s.Spec.Config = append(s.Spec.Config, core.EnvVar{Name: "ELASTIC_USER", Value: "elastic"})
			s.Spec.Config = append(s.Spec.Config, core.EnvVar{Name: "ELASTIC_PASSWORD", Value: "changeme"})
		} else {
			usersecret := core.Secret{}
			if err := r.Client.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: user.SecretName}, &usersecret); err != nil && !apierrors.IsNotFound(err) {
				log.Info("fail get usersecret ")
			}
			for k, v := range usersecret.Data {
				if k == "username" {
					s.Spec.Config = append(s.Spec.Config, core.EnvVar{Name: "ELASTIC_USER", Value: string(v)})
				} else if k == "password" {
					s.Spec.Config = append(s.Spec.Config, core.EnvVar{Name: "ELASTIC_PASSWORD", Value: string(v)})
				}
			}
		}
	}
	if tls {
		s.Spec.ServiceName = "skywalking-storage"
	} else {
		s.Spec.ServiceName = s.Name + "-" + s.Spec.Type
	}
	if s.Spec.ResourceCnfig.Limit == "" && s.Spec.ResourceCnfig.Requests == "" {
		s.Spec.ResourceCnfig.Limit, s.Spec.ResourceCnfig.Requests = "1000m", "100m"
	}

	setDefaultJavaOpts := true
	for _, envVar := range s.Spec.Config {
		if envVar.Name == "ES_JAVA_OPTS" {
			setDefaultJavaOpts = false
		}
	}
	if setDefaultJavaOpts {
		s.Spec.Config = append(s.Spec.Config, core.EnvVar{Name: "ES_JAVA_OPTS", Value: "-Xms1g -Xmx1g"})
	}
	s.Spec.Config = append(s.Spec.Config, core.EnvVar{Name: "discovery.seed_hosts", Value: s.Spec.ServiceName})
	clusterInitialMasterNodes := make([]string, s.Spec.Instances)
	for i := 0; i < int(s.Spec.Instances); i++ {
		clusterInitialMasterNodes[i] = s.Name + "-elasticsearch-" + strconv.Itoa(i)
	}
	s.Spec.Config = append(s.Spec.Config, core.EnvVar{Name: "cluster.initial_master_nodes", Value: strings.Join(clusterInitialMasterNodes, ",")})
}

func (r *StorageReconciler) createCert(ctx context.Context, log logr.Logger, s *operatorv1alpha1.Storage) {
	clientset, err := kubeclient.NewForConfig(r.RestConfig)
	if err != nil {
		return
	}
	existSecret, err := clientset.CoreV1().Secrets(s.Namespace).Get(ctx, "skywalking-storage", metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Info("fail get skywalking-storage secret")
		return
	}
	if existSecret.Name != "" {
		_, verifyCert, decodeErr := pkcs12.Decode(existSecret.Data["storage.p12"], "")
		if decodeErr != nil {
			log.Info("decode storage.p12 error")
			return
		}
		if time.Until(verifyCert.NotAfter).Hours() < 24 {
			log.Info("storage cert will expire,the storage cert will re-generate")
		} else {
			return
		}
	}
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Info("fail generate privatekey")
		return
	}
	subj := pkix.Name{
		CommonName:         "skywalking-storage",
		Country:            []string{"CN"},
		Province:           []string{"ZJ"},
		Locality:           []string{"HZ"},
		Organization:       []string{"Skywalking"},
		OrganizationalUnit: []string{"Skywalking"},
	}
	asn1Subj, err := asn1.Marshal(subj.ToRDNSequence())
	if err != nil {
		return
	}
	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		log.Info("fail create certificaterequest")
		return
	}
	buffer := new(bytes.Buffer)
	err = pem.Encode(buffer, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	if err != nil {
		log.Info("fail encode CERTIFICATE REQUEST")
		return
	}
	singername := "kubernetes.io/legacy-unknown"
	request := certv1beta1.CertificateSigningRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CertificateSigningRequest",
			APIVersion: "certificates.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "storage-csr",
		},
		Spec: certv1beta1.CertificateSigningRequestSpec{
			Groups:     []string{"system:authenticated"},
			Request:    buffer.Bytes(),
			SignerName: &singername,
			Usages:     []certv1beta1.KeyUsage{certv1beta1.UsageClientAuth},
		},
	}
	err = clientset.CertificatesV1beta1().CertificateSigningRequests().Delete(ctx, "storage-csr", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Info("fail delete csr")
		return
	}
	csr, err := clientset.CertificatesV1beta1().CertificateSigningRequests().Create(ctx, &request, metav1.CreateOptions{})
	if err != nil {
		log.Info("fail create csr")
		return
	}
	condition := certv1beta1.CertificateSigningRequestCondition{
		Type:    "Approved",
		Reason:  "ApprovedBySkywalkingStorage",
		Message: "Approved by skywalking storage controller",
	}
	csr.Status.Conditions = append(csr.Status.Conditions, condition)
	updateapproval, err := clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(ctx, csr, metav1.UpdateOptions{})
	if err != nil {
		log.Info("fail update approval", "result", updateapproval, "err", err)
		return
	}
	for {
		csr, err = clientset.CertificatesV1beta1().CertificateSigningRequests().Get(ctx, "storage-csr", metav1.GetOptions{})
		if err != nil {
			log.Info("fail get storage-csr")
			return
		}
		if csr.Status.Certificate != nil {
			break
		}
	}
	block, _ := pem.Decode(csr.Status.Certificate)
	cert, err := x509.ParseCertificate(block.Bytes)

	if err != nil {
		log.Info("fail parse certificate")
		return
	}
	p12, err := pkcs12.Encode(rand.Reader, key, cert, nil, "skywalking")

	if err != nil {
		log.Info("fail encode pkcs12")
		return
	}
	err = clientset.CoreV1().Secrets(s.Namespace).Delete(ctx, "skywalking-storage", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		log.Info("fail delete secret skywalking-storage")
		return
	}
	data := make(map[string][]byte)
	data["storage.p12"] = p12
	secret := core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "skywalking-storage",
			Namespace: s.Namespace,
		},
		Data: data,
		Type: "Opaque",
	}
	storageTLSSecret, err := clientset.CoreV1().Secrets(s.Namespace).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		log.Info("fail create secret skywalking-storage")
		return
	}
	log.Info("success create secret skywalking-storage", "secret-name", storageTLSSecret.Name)
}

func (r *StorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Storage{}).
		Owns(&core.Service{}).
		Complete(r)
}
