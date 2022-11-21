// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"net/http"
	"net/http/pprof"
	"os"

	akov1alpha1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha1"
	"go.uber.org/zap/zapcore"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers"

	kapppkgiv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"
	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	runv1alpha3 "github.com/vmware-tanzu/tanzu-framework/apis/run/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = log.Log
)

func initLog() {
	f := func(ecfg *zapcore.EncoderConfig) {
		ecfg.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	ctrl.SetLogger(zap.New(zap.UseDevMode(true),
		zap.ConsoleEncoder(zap.EncoderConfigOption(f))))
}

func init() {
	initLog()
	// ignoring errors
	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = akoov1alpha1.AddToScheme(scheme)
	_ = akov1alpha1.AddToScheme(scheme)
	_ = runv1alpha3.AddToScheme(scheme)
	_ = kapppkgiv1alpha1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var profilerAddress string
	flag.StringVar(&metricsAddr, "metrics-addr", "localhost:8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&profilerAddress, "profiler-addr", "", "Bind address to expose the pprof profiler")
	flag.Parse()

	if profilerAddress != "" {
		setupLog.Info(
			"Profiler listening for requests",
			"profiler-addr", profilerAddress)
		go runProfiler(profilerAddress)
	}
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Port:               9443,
		ClientDisableCacheFor: []client.Object{
			&corev1.ConfigMap{},
			&corev1.Secret{},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = controllers.SetupReconcilers(mgr)
	if err != nil {
		setupLog.Error(err, "Unable to setup reconcilers")
		os.Exit(1)
	}

	//setup webhook here
	if !ako_operator.IsBootStrapCluster() {
		err = (&akoov1alpha1.AKODeploymentConfig{}).SetupWebhookWithManager(mgr)
		if err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "AKODeploymentConfig")
			os.Exit(1)
		}
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func runProfiler(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		setupLog.Error(err, "unable to start listening")
	}
}
