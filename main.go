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

package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/allenhaozi/webhook/api/common"
	webhookv1 "github.com/allenhaozi/webhook/api/v1"
	webhookv1alpha1 "github.com/allenhaozi/webhook/api/v1alpha1"
	"github.com/allenhaozi/webhook/controllers"
	"github.com/allenhaozi/webhook/pkg/manager"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("webhook setup main")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(webhookv1.AddToScheme(scheme))
	utilruntime.Must(webhookv1alpha1.AddToScheme(scheme))

	// add argoproj workflow
	// utilruntime.Must(argoworkflowv1alpha1.AddToScheme(scheme))
	// local dummy workflow
	utilruntime.Must(webhookv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	setupLog.Info("Starting webhook main logic")
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var certDir string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs", "webhook certificate.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("start new manager with get k8s config")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		CertDir:                certDir,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "10b03d2d.github.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	setupLog.Info("start webhook controller")
	if err = (&controllers.MetaWebHookReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, setupLog); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MetaWebHook")
		os.Exit(1)
	}

	cfg, _ := ctrl.GetConfig()
	client, _ := client.New(cfg, client.Options{})
	// certificate manager
	// generate certificate
	// 1. store it in secret
	// 2. save in local pod path certDir
	certManager := manager.NewCertificateManager(client, setupLog, certDir)

	ns := ""
	if v, ok := os.LookupEnv(common.MyPodNamespace); ok {
		ns = v
	} else {
		setupLog.Error(err, "get environment failure")
		os.Exit(1)
	}

	certContext, err := certManager.GenerateCertificate(ns, common.WebHookName)
	if err != nil {
		setupLog.Error(err, "generate certification failure")
		os.Exit(1)
	}

	if err = (&controllers.MutatingWebhookConfigurationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, setupLog, certContext); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MutatingWebhookConfigurationReconciler")
		os.Exit(1)
	}

	// webhook register

	setupLog.Info("start webhook server and register it with SetupWebhookWithManager")

	if err = (&webhookv1.MetaWebHook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "MetaWebHook")
		os.Exit(1)
	}

	hookServer := mgr.GetWebhookServer()

	hookServer.Register("/mutate-v1alpha1-argoworkflow", &webhook.Admission{Handler: &webhookv1alpha1.ArgoWorkflowHandler{Client: mgr.GetClient(), Log: setupLog}})

	//+kubebuilder:scaffold:builder

	setupLog.Info("health check")
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	setupLog.Info("read check")
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
