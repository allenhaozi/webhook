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
	"reflect"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/allenhaozi/webhook/api/common"
)

// MetaWebHookReconciler reconciles a MetaWebHook object
type MutatingWebhookConfigurationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	CaCert []byte
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfiguration/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfiguration/finalizers,verbs=update
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MetaWebHook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MutatingWebhookConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var m admissionv1.MutatingWebhookConfiguration
	if err := r.Get(ctx, req.NamespacedName, &m); err != nil {
		r.Log.Error(err, "got error")
		return ctrl.Result{}, err
	}

	r.Log.Info("received mutatingwebhookconfiguration data", "MutatingWebhookConfiguration", req.NamespacedName)

	if err := r.patchCaBundle(&m); err != nil {
		r.Log.Error(err, "fail to patch CABundle to mutatingWebHookConfiguration")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MutatingWebhookConfigurationReconciler) patchCaBundle(m *admissionv1.MutatingWebhookConfiguration) error {
	ctx := context.Background()

	current := m.DeepCopy()
	for i := range m.Webhooks {
		m.Webhooks[i].ClientConfig.CABundle = r.CaCert
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
func (r *MutatingWebhookConfigurationReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger, caCert []byte) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&admissionv1.MutatingWebhookConfiguration{}).
		WithEventFilter(filterByWebhookName).
		Complete(
			NewMutatingWebhookConfigurationReconciler(mgr, l, caCert),
		)
}

func NewMutatingWebhookConfigurationReconciler(mgr ctrl.Manager, l logr.Logger, caCert []byte) *MutatingWebhookConfigurationReconciler {
	r := &MutatingWebhookConfigurationReconciler{}
	r.Client = mgr.GetClient()
	r.Log = l
	r.CaCert = caCert
	return r
}
