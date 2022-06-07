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
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	operatorcontroller "github.com/apache/skywalking-swck/operator/controllers/operator"
	operatorcontrollers "github.com/apache/skywalking-swck/operator/controllers/operator"
	"github.com/apache/skywalking-swck/operator/pkg/operator/injector"
	"github.com/apache/skywalking-swck/operator/pkg/operator/manifests"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err = (&operatorcontroller.OAPServerReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		FileRepo: manifests.NewRepo("oapserver"),
		Recorder: mgr.GetEventRecorderFor("oapserver-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OAPServer")
		os.Exit(1)
	}
	if err = (&operatorcontroller.UIReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		FileRepo: manifests.NewRepo("ui"),
		Recorder: mgr.GetEventRecorderFor("oapserver-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "UI")
		os.Exit(1)
	}
	if err = (&operatorcontroller.FetcherReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		FileRepo: manifests.NewRepo("fetcher"),
		Recorder: mgr.GetEventRecorderFor("fetcher-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Fetcher")
		os.Exit(1)
	}
	if err = (&operatorcontroller.StorageReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		FileRepo:   manifests.NewRepo("storage"),
		RestConfig: mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Storage")
		os.Exit(1)
	}
	if err = (&operatorcontroller.ConfigMapReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		FileRepo: manifests.NewRepo("injector"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConfigMap")
		os.Exit(1)
	}
	if err = (&operatorcontroller.JavaAgentReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		FileRepo: manifests.NewRepo("injector"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "JavaAgent")
		os.Exit(1)
	}

	if err = (&operatorcontrollers.SatelliteReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		FileRepo: manifests.NewRepo("satellite"),
		Recorder: mgr.GetEventRecorderFor("satellite-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Satellite")
		os.Exit(1)
	}
	if err = (&operatorcontrollers.SwAgentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SwAgent")
		os.Exit(1)
	}
	if err = (&operatorcontrollers.OAPServerConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OAPServerConfig")
		os.Exit(1)
	}
	if err = (&operatorcontrollers.OAPServerDynamicConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OAPServerDynamicConfig")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder
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
			setupLog.Error(err, "unable to create webhook", "webhook", "Storage")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.JavaAgent{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "JavaAgent")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.Satellite{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Satellite")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.SwAgent{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "SwAgent")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.OAPServerConfig{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OAPServerConfig")
			os.Exit(1)
		}
		if err = (&operatorv1alpha1.OAPServerDynamicConfig{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OAPServerDynamicConfig")
			os.Exit(1)
		}
		// register a webhook to enable the java agent injector
		setupLog.Info("registering /mutate-v1-pod webhook")
		mgr.GetWebhookServer().Register("/mutate-v1-pod",
			&webhook.Admission{
				Handler: &injector.JavaagentInjector{Client: mgr.GetClient()}})
		setupLog.Info("/mutate-v1-pod webhook is registered")

		if err := mgr.AddHealthzCheck("healthz", mgr.GetWebhookServer().StartedChecker()); err != nil {
			setupLog.Error(err, "unable to set up health check for webhook")
			os.Exit(1)
		}
		if err := mgr.AddReadyzCheck("readyz", mgr.GetWebhookServer().StartedChecker()); err != nil {
			setupLog.Error(err, "unable to set up ready check for webhook")
			os.Exit(1)
		}
	} else {
		if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
			setupLog.Error(err, "unable to set up ready check")
		}
		if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
			setupLog.Error(err, "unable to set up ready check")
		}
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
