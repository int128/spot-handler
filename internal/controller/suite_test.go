/*
Copyright 2025.

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

package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ktesting "k8s.io/utils/clock/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	spothandlerv1 "github.com/int128/spot-handler/api/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sClient ctrlclient.Client
var mockSQSClient mockSQSClientType
var fakeNow = time.Date(2021, 7, 1, 1, 1, 1, 0, time.UTC)
var spotInterruptedPodTerminationReconcilerClock = ktesting.NewFakeClock(fakeNow)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	ctrllog.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	err := spothandlerv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}
	DeferCleanup(func() {
		By("tearing down the test environment")
		Expect(testEnv.Stop()).To(Succeed())
	})

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	Expect((&QueueReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		SQSClient: &mockSQSClient,
	}).SetupWithManager(mgr)).To(Succeed())

	Expect((&SpotInterruptionReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Clock:  ktesting.NewFakePassiveClock(fakeNow),
	}).SetupWithManager(mgr)).To(Succeed())

	Expect((&SpotInterruptedNodeReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Clock:  ktesting.NewFakePassiveClock(fakeNow),
	}).SetupWithManager(mgr)).To(Succeed())

	Expect((&SpotInterruptedPodReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Clock:  ktesting.NewFakePassiveClock(fakeNow),
	}).SetupWithManager(mgr)).To(Succeed())

	Expect((&SpotInterruptedPodTerminationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Clock:  spotInterruptedPodTerminationReconcilerClock,
	}).SetupWithManager(mgr)).To(Succeed())

	ctx, cancel := context.WithCancel(context.TODO())
	DeferCleanup(func() {
		cancel()
	})
	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		ctrllog.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
