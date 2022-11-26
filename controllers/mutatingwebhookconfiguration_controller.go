/*
Copyright 2022.

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

package controllers

import (
	"context"
	"os"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/allenhaozi/webhook/api/common"
	webhookutils "github.com/allenhaozi/webhook/pkg/utils"
)

// MetaWebHookReconciler reconciles a MetaWebHook object
type MutatingWebhookConfigurationReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	CertDir string
	Log     logr.Logger
}

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfiguration/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfiguration/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MutatingWebhookConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var m admissionv1.MutatingWebhookConfiguration
	if err := r.Get(ctx, req.NamespacedName, &m); err != nil {
		r.Log.Error(err, "got error")
		return ctrl.Result{}, err
	}

	r.Log.Info("received mutatingwebhookconfiguration data", "MutatingWebhookConfiguration", req.NamespacedName)

	// certificate logic
	certContext, err := r.genCertificates(req.NamespacedName)
	if err != nil {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, err
	}

	if err := r.patchCaBundle(&m, certContext); err != nil {
		r.Log.Error(err, "fail to patch CABundle to mutatingWebHookConfiguration")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MutatingWebhookConfigurationReconciler) genCertificates(objectKey apitypes.NamespacedName) (*common.CertContext, error) {
	// if secret key exists, use the secret values
	// won't create new certificates
	ns := ""
	if v, ok := os.LookupEnv(common.MyPodNamespace); ok {
		ns = v
	} else {
		err := errors.Errorf("can not get %s environment variable", common.MyPodNamespace)
		r.Log.Error(err, "get namespace environment failure")
	}

	ctx := context.Background()

	secret := &corev1.Secret{}
	err := r.Get(ctx, objectKey, secret)

	// if secret not found
	if kerrors.IsNotFound(err) {
		// trigger generate ca logic
		certContext, err := webhookutils.GenerateCert(ns, common.WebHookName)
		if err != nil {
			r.Log.Error(err, "get namespace environment failure")
		}
		// write certificate file to local directory
		if err := certContext.WriteCertFileToLocal(r.CertDir); err != nil {
			r.Log.Error(err, "write certificate file to local failure")
		}
		// persist certificate to secret
		secret := certContext.ComposeSecrets(objectKey.Namespace, objectKey.Name)
		// TODO: use patch
		if err := r.Create(ctx, secret); err != nil {
			return nil, errors.Wrap(err, "create secret failure")
		}
		return certContext, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "generate certificate failure")
	}

	// certificate secret exists, use the values
	certContext, err := common.GenerateCertBySecret(secret)
	if err != nil {
		return nil, errors.Wrap(err, "parse ")
	}

	return certContext, nil
}

func (r *MutatingWebhookConfigurationReconciler) patchCaBundle(m *admissionv1.MutatingWebhookConfiguration, certContext *common.CertContext) error {
	ctx := context.Background()

	current := m.DeepCopy()
	for i := range m.Webhooks {
		m.Webhooks[i].ClientConfig.CABundle = certContext.SigningCert
	}

	if reflect.DeepEqual(m.Webhooks, current.Webhooks) {
		r.Log.Info("no need to patch the MutatingWebhookConfiguration", "name", m.GetName())
		return nil
	}

	if err := r.Patch(ctx, m, client.MergeFrom(current)); err != nil {
		r.Log.Error(err, "fail to patch CABundle to mutatingWebHook", "name", m.GetName())
		return err
	}

	r.Log.Info("finished patch MutatingWebhookConfiguration caBundle", "name", m.GetName())

	return nil
}

// add

var filterByWebhookName = &predicate.Funcs{
	CreateFunc: createPredicate,
	UpdateFunc: updatePredicate,
}

func createPredicate(e event.CreateEvent) bool {
	obj, ok := e.Object.(*admissionv1.MutatingWebhookConfiguration)
	if !ok {
		return false
	}

	if obj.GetName() == common.WebHookName {
		return true
	}

	return false
}

func updatePredicate(e event.UpdateEvent) bool {
	obj, ok := e.ObjectOld.(*admissionv1.MutatingWebhookConfiguration)
	if !ok {
		return false
	}

	if obj.GetName() == common.WebHookName {
		return true
	}

	return false
}

// MutatingWebhookConfiguration
// SetupWithManager sets up the controller with the Manager.
func (r *MutatingWebhookConfigurationReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger, certDir string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&admissionv1.MutatingWebhookConfiguration{}).
		WithEventFilter(filterByWebhookName).
		Complete(
			NewMutatingWebhookConfigurationReconciler(mgr, l, certDir),
		)
}

func NewMutatingWebhookConfigurationReconciler(mgr ctrl.Manager, l logr.Logger, certDir string) *MutatingWebhookConfigurationReconciler {
	r := &MutatingWebhookConfigurationReconciler{}
	r.Client = mgr.GetClient()
	r.Log = l
	r.CertDir = certDir
	return r
}
