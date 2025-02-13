/*
Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.

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
	"time"

	commonv1 "go.amzn.com/sagemaker/sagemaker-k8s-operator/api/v1/common"
	"go.amzn.com/sagemaker/sagemaker-k8s-operator/controllers/batchtransformjob"
	"go.amzn.com/sagemaker/sagemaker-k8s-operator/controllers/endpointconfig"
	"go.amzn.com/sagemaker/sagemaker-k8s-operator/controllers/hosting"
	"go.amzn.com/sagemaker/sagemaker-k8s-operator/controllers/hyperparametertuningjob"
	"go.amzn.com/sagemaker/sagemaker-k8s-operator/controllers/model"
	"go.amzn.com/sagemaker/sagemaker-k8s-operator/controllers/trainingjob"

	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	commonv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	const jobPollInterval = "5s"

	err = trainingjob.
		NewTrainingJobReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("TrainingJob"),
			parseDurationOrPanic(jobPollInterval)).
		SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TrainingJob")
		os.Exit(1)
	}

	err = hyperparametertuningjob.
		NewHyperparameterTuningJobReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("HyperparameterTuningJobReconciler"),
			parseDurationOrPanic(jobPollInterval)).
		SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HyperparameterTuningJobReconciler")
		os.Exit(1)
	}

	err = hosting.
		NewHostingDeploymentReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("HostingDeployment"),
			parseDurationOrPanic("5s")).
		SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HostingDeployment")
		os.Exit(1)
	}

	err = model.
		NewModelReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("Model"),
			// TODO change to 1m for release. 5s is only to catch bugs during development.
			parseDurationOrPanic("5s")).
		SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Model")
		os.Exit(1)
	}

	err = endpointconfig.
		NewEndpointConfigReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("EndpointConfig"),
			// TODO change to 1m for release. 5s is only to catch bugs during development.
			parseDurationOrPanic("5s")).
		SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EndpointConfig")
		os.Exit(1)
	}

	err = batchtransformjob.
		NewBatchTransformJobReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("BatchTransformJob"),
			parseDurationOrPanic(jobPollInterval)).
		SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BatchTransformJob")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func parseDurationOrPanic(duration string) time.Duration {
	if parsed, err := time.ParseDuration(duration); err != nil {
		panic("Unable to parse duration: " + duration)
	} else {
		return parsed
	}
}
