/*
Copyright 2018 The Kubernetes Authors.

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

package controller_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"

	"github.com/cjhpaul/controller-runtime/pkg/client"
	"github.com/cjhpaul/controller-runtime/pkg/controller"
	"github.com/cjhpaul/controller-runtime/pkg/manager"
	"github.com/cjhpaul/controller-runtime/pkg/reconcile"
	"github.com/cjhpaul/controller-runtime/pkg/runtime/inject"
)

var _ = Describe("controller.Controller", func() {
	var stop chan struct{}

	rec := reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
		return reconcile.Result{}, nil
	})
	BeforeEach(func() {
		stop = make(chan struct{})
	})

	AfterEach(func() {
		close(stop)
	})

	Describe("New", func() {
		It("should return an error if Name is not Specified", func(done Done) {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())
			c, err := controller.New("", m, controller.Options{Reconciler: rec})
			Expect(c).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must specify Name for Controller"))

			close(done)
		})

		It("should return an error if Reconciler is not Specified", func(done Done) {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			c, err := controller.New("foo", m, controller.Options{})
			Expect(c).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must specify Reconciler"))

			close(done)
		})

		It("NewController should return an error if injecting Reconciler fails", func(done Done) {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			c, err := controller.New("foo", m, controller.Options{Reconciler: &failRec{}})
			Expect(c).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected error"))

			close(done)
		})

		It("should not return an error if two controllers are registered with different names", func(done Done) {
			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			c1, err := controller.New("c1", m, controller.Options{Reconciler: rec})
			Expect(err).NotTo(HaveOccurred())
			Expect(c1).ToNot(BeNil())

			c2, err := controller.New("c2", m, controller.Options{Reconciler: rec})
			Expect(err).NotTo(HaveOccurred())
			Expect(c2).ToNot(BeNil())

			close(done)
		})

		It("should not leak goroutines when stopped", func() {
			currentGRs := goleak.IgnoreCurrent()

			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			_, err = controller.New("new-controller", m, controller.Options{Reconciler: rec})
			Expect(err).NotTo(HaveOccurred())

			s := make(chan struct{})
			close(s)

			Expect(m.Start(s)).To(Succeed())

			// force-close keep-alive connections.  These'll time anyway (after
			// like 30s or so) but force it to speed up the tests.
			clientTransport.CloseIdleConnections()
			Eventually(func() error { return goleak.Find(currentGRs) }).Should(Succeed())
		})

		It("should not create goroutines if never started", func() {
			currentGRs := goleak.IgnoreCurrent()

			m, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())

			_, err = controller.New("new-controller", m, controller.Options{Reconciler: rec})
			Expect(err).NotTo(HaveOccurred())

			// force-close keep-alive connections.  These'll time anyway (after
			// like 30s or so) but force it to speed up the tests.
			clientTransport.CloseIdleConnections()
			Eventually(func() error { return goleak.Find(currentGRs) }).Should(Succeed())
		})
	})
})

var _ reconcile.Reconciler = &failRec{}
var _ inject.Client = &failRec{}

type failRec struct{}

func (*failRec) Reconcile(context.Context, reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (*failRec) InjectClient(client.Client) error {
	return fmt.Errorf("expected error")
}
