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

package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	operatorv1alpha1 "github.com/apache/skywalking-swck/apis/operator/v1alpha1"
	operatorcontroller "github.com/apache/skywalking-swck/controllers/operator"
	"github.com/apache/skywalking-swck/pkg/operator/repo"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = operatorv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var dev bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&dev, "dev", false, "Enable development mode")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = dev
	}))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "v1alpha1.swck",
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&operatorcontroller.OAPServerReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OAPServer"),
		Scheme:   mgr.GetScheme(),
		FileRepo: repo.NewRepo("oapserver"),
		Recorder: mgr.GetEventRecorderFor("oapserver-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OAPServer")
		os.Exit(1)
	}
	if err = (&operatorcontroller.UIReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("UI"),
		Scheme:   mgr.GetScheme(),
		FileRepo: repo.NewRepo("ui"),
		Recorder: mgr.GetEventRecorderFor("oapserver-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "UI")
		os.Exit(1)
	}

	if err = (&operatorcontroller.FetcherReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("Fetcher"),
		Scheme:   mgr.GetScheme(),
		FileRepo: repo.NewRepo("fetcher"),
		Recorder: mgr.GetEventRecorderFor("fetcher-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Fetcher")
		os.Exit(1)
	}

	if err = (&operatorcontroller.StorageReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("Storage"),
		Scheme:   mgr.GetScheme(),
		FileRepo: repo.NewRepo("storage"),
		Recorder: mgr.GetEventRecorderFor("storage-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Storage")
		os.Exit(1)
	}

	if err = (&operatorcontroller.ConfigMapReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ConfigMap"),
		Scheme:   mgr.GetScheme(),
		FileRepo: repo.NewRepo("injector"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConfigMap")
		os.Exit(1)
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&operatorv1alpha1.OAPServer{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OAPServer")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.UI{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "UI")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.Fetcher{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Fetcher")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.Storage{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "storage")
			os.Exit(1)
		}
		// register a webhook to enable the agent injector，
		setupLog.Info("registering /mutate-v1-pod webhook")
		mgr.GetWebhookServer().Register("/mutate-v1-pod",
			&webhook.Admission{
				Handler: &operatorv1alpha1.Javaagent{Client: mgr.GetClient()}})
		setupLog.Info("/mutate-v1-pod webhook is registered")
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
