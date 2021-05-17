// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"github.com/projectcontour/contour/internal/contour"
	"github.com/projectcontour/contour/internal/dag"
	"github.com/projectcontour/contour/internal/k8s"
	"github.com/projectcontour/contour/internal/xdscache"
	"path/filepath"
	controller_config "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	gatewayv1alpha1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// Define utility constants for object names, testing timeouts/durations intervals, etc.
const (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

var (
	log logrus.FieldLogger
	testEnv *envtest.Environment

	gc = &gatewayv1alpha1.GatewayClass{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: gatewayv1alpha1.GatewayClassSpec{
			Controller: "projectcontour.io/projectcontour/contour",
		},
	}

	ctx = context.Background()
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	By("Bootstrapping the test environment")
	contourCRDs := filepath.Join("..", "..", "examples", "contour", "01-crds.yaml")
	gatewayCRDs := filepath.Join("..", "..", "examples", "gateway", "00-crds.yaml")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{contourCRDs, gatewayCRDs},
	}

	cliCfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cliCfg).ToNot(BeNil())

	// Create the controller manager.
	mgr, err := manager.New(controller_config.GetConfigOrDie(), manager.Options{})
	Expect(err).ToNot(HaveOccurred())

	// Before we can build the event handler, we need to initialize the converter we'll
	// use to convert from Unstructured.
	converter, err = k8s.NewUnstructuredConverter()
	Expect(err).ToNot(HaveOccurred())

	// Build the core Kubernetes event handler.
	eventHandler := &contour.EventHandler{
		HoldoffDelay:    100 * time.Millisecond,
		HoldoffMaxDelay: 500 * time.Millisecond,
		Observer:        dag.ComposeObservers(append(xdscache.ObserversOf(resources), snapshotHandler)...),
		Builder:         getDAGBuilder(ctx, clients, clientCert, fallbackCert, log),
		FieldLogger:     log.WithField("context", "contourEventHandler"),
	}

	// Create and register the gatewayclass controller with the manager.
	id := "projectcontour.io/projectcontour/contour"
	_, err = NewGatewayClassController(mgr, nil, log.WithField("context", "gatewayclass-controller"), id)
	Expect(err).ToNot(HaveOccurred())

	// Start the manager.
	go func() {
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("Expecting the test environment teardown to complete")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
