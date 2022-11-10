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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "github.com/allenhaozi/webhook/api/v1"
)

// MetaWebHookReconciler reconciles a MetaWebHook object
type MetaWebHookReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=meta.github.com,resources=metawebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=meta.github.com,resources=metawebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meta.github.com,resources=metawebhooks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MetaWebHook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MetaWebHookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var webhook metav1.MetaWebHook
	if err := r.Get(ctx, req.NamespacedName, &webhook); err != nil {
		r.Log.Error(err, "got error")
		return ctrl.Result{}, err
	}

	r.Log.Info("received webhook data", "webhook", webhook)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MetaWebHookReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&metav1.MetaWebHook{}).
		Complete(
			NewMetaWebHookReconciler(mgr, l),
		)
}

func NewMetaWebHookReconciler(mrg ctrl.Manager, l logr.Logger) *MetaWebHookReconciler {
	r := &MetaWebHookReconciler{}
	r.Log = l
	r.Client = mrg.GetClient()
	return r
}
