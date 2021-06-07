// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	//nolint
	. "github.com/onsi/ginkgo"

	//nolint
	. "github.com/onsi/gomega"

	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/aviclient"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToScheme is the function TestSuite calls to register schemes for a manager
type AddToSchemeFunc func(*runtime.Scheme) error

// AddToManagerFunc is the function controller calls to register itself with the
// manager passed in
type AddToManagerFunc func(ctrlmgr.Manager) error

// Reconciler is a base type for builder's reconcilers
type Reconciler interface{}

// NewReconcilerFunc is a base type for functions that return a reconciler
type NewReconcilerFunc func() Reconciler

func init() {
	//klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)
	logf.SetLogger(klogr.New())
}

// TestSuite is used for unit and integration testing builder. Each TestSuite
// contains one independent test environment and a controller manager
type TestSuite struct {
	context.Context
	flags testFlags

	addToManagerFn AddToManagerFunc

	envTest               envtest.Environment
	integrationTestClient client.Client
	config                *rest.Config

	manager             manager.Manager
	addToScheme         AddToSchemeFunc
	managerDone         chan struct{}
	managerRunning      bool
	managerRunningMutex sync.Mutex
}

func (s *TestSuite) GetEnvTestConfg() *rest.Config {
	return s.config
}

func (s *TestSuite) GetManager() manager.Manager {
	return s.manager
}

// NewTestSuiteForController returns a new test suite used for integration test
func NewTestSuiteForController(addToManagerFn AddToManagerFunc, addToSchemeFn AddToSchemeFunc, crdpaths ...string) *TestSuite {

	testSuite := &TestSuite{
		Context:        context.Background(),
		addToScheme:    addToSchemeFn,
		addToManagerFn: addToManagerFn,
	}
	testSuite.init(crdpaths)

	return testSuite
}

// NewTestSuiteForController returns a new test suite used for integration test
func NewTestSuiteForReconciler(addToManagerFn AddToManagerFunc, addToSchemeFn AddToSchemeFunc, crdpaths ...string) *TestSuite {

	testSuite := &TestSuite{
		Context:        context.Background(),
		addToScheme:    addToSchemeFn,
		addToManagerFn: addToManagerFn,
	}
	testSuite.init(crdpaths)

	return testSuite
}

func (s *TestSuite) init(crdpaths []string, additionalAPIServerFlags ...string) {
	// Initialize the test flags.
	s.flags = flags

	if s.flags.IntegrationTestsEnabled {
		if s.addToManagerFn == nil {
			panic("addToManagerFn is nil")
		}

		apiServerFlags := append([]string{"--allow-privileged=true"}, envtest.DefaultKubeAPIServerFlags...)
		if len(additionalAPIServerFlags) > 0 {
			apiServerFlags = append(apiServerFlags, additionalAPIServerFlags...)
		}

		crdpaths = append(crdpaths, filepath.Join(s.flags.RootDir, "config", "crd", "bases"))
		s.envTest = envtest.Environment{
			CRDDirectoryPaths:  crdpaths,
			KubeAPIServerFlags: apiServerFlags,
		}
	}
}

// Register should be invoked by the function to which *testing.T is passed.
//
// Use runUnitTestsFn to pass a function that will be invoked if unit testing
// is enabled with Describe("Unit tests", runUnitTestsFn).
//
// Use runIntegrationTestsFn to pass a function that will be invoked if
// integration testing is enabled with
// Describe("Unit tests", runIntegrationTestsFn).
func (s *TestSuite) Register(t *testing.T, name string, runIntegrationTestsFn, runUnitTestsFn func()) {
	RegisterFailHandler(Fail)

	if runIntegrationTestsFn == nil {
		s.flags.IntegrationTestsEnabled = false
	}
	if runUnitTestsFn == nil {
		s.flags.UnitTestsEnabled = false
	}

	if s.flags.IntegrationTestsEnabled {
		Describe("Integration tests", runIntegrationTestsFn)
	}
	if s.flags.UnitTestsEnabled {
		Describe("Unit tests", runUnitTestsFn)
	}

	if s.flags.IntegrationTestsEnabled {
		SetDefaultEventuallyTimeout(time.Second * 10)
		SetDefaultEventuallyPollingInterval(time.Second)
		RunSpecsWithDefaultAndCustomReporters(t, name, []Reporter{printer.NewlineReporter{}})
	} else if s.flags.UnitTestsEnabled {
		RunSpecs(t, name)
	}
}

// BeforeSuite should be invoked by ginkgo.BeforeSuite.
func (s *TestSuite) BeforeSuite() {
	if s.flags.IntegrationTestsEnabled {
		s.beforeSuiteForIntegrationTesting()
	}
}

func (s *TestSuite) beforeSuiteForIntegrationTesting() {
	By("bootstrapping test environment", func() {
		var err error
		s.config, err = s.envTest.Start()
		Expect(err).ToNot(HaveOccurred())
		Expect(s.config).ToNot(BeNil())
	})

	By("setting up a new manager", func() {
		s.createManager()
	})

	By("starting the manager", func() {
		s.startManager()
	})
}

// Create a new Manager with default values
func (s *TestSuite) createManager() {
	var err error
	s.managerDone = make(chan struct{})

	// Create a new Scheme for each controller. Don't use a global scheme otherwise manager reset
	// will try to reinitialize the global scheme which causes errors
	managerScheme := runtime.NewScheme()
	// Register schemes using the passed function
	err = s.addToScheme(managerScheme)
	Expect(err).NotTo(HaveOccurred())

	s.manager, err = manager.New(s.config, manager.Options{
		Scheme:             managerScheme,
		MetricsBindAddress: "0",
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			syncPeriod := 1 * time.Second
			opts.Resync = &syncPeriod
			return cache.New(config, opts)
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(s.manager).ToNot(BeNil())

	// Register controllers using the passed function
	err = s.addToManagerFn(s.manager)
	Expect(err).NotTo(HaveOccurred())
	s.integrationTestClient = s.manager.GetClient()
}

// Starts the manager and sets managerRunning
func (s *TestSuite) startManager() {
	go func() {
		defer GinkgoRecover()

		s.setManagerRunning(true)
		Expect(s.manager.Start(s.managerDone)).ToNot(HaveOccurred())
		s.setManagerRunning(false)
	}()
}

// Set a flag to indicate that the manager is running or not
func (s *TestSuite) setManagerRunning(isRunning bool) {
	s.managerRunningMutex.Lock()
	s.managerRunning = isRunning
	s.managerRunningMutex.Unlock()
}

/*// Returns true if the manager is running, false otherwise*/
func (s *TestSuite) getManagerRunning() bool {
	var result bool
	s.managerRunningMutex.Lock()
	result = s.managerRunning
	s.managerRunningMutex.Unlock()
	return result
}

// AfterSuite should be invoked by ginkgo.AfterSuite.
func (s *TestSuite) AfterSuite() {
	if s.flags.IntegrationTestsEnabled {
		s.afterSuiteForIntegrationTesting()
	}
}

func (s *TestSuite) afterSuiteForIntegrationTesting() {
	By("tearing down the test environment", func() {
		s.stopManager()
		Expect(s.envTest.Stop()).To(Succeed())
	})
}

func (s *TestSuite) stopManager() {
	close(s.managerDone)
	Eventually(s.getManagerRunning).Should((BeFalse()))
}

var FakeAvi *aviclient.FakeAviClient
