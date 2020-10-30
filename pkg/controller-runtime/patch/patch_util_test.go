// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package patch_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"

	. "github.com/onsi/ginkgo" // nolint
	. "github.com/onsi/gomega" // nolint

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/controller-runtime/patch"
)

var _ = Describe("CreateOrPatch", func() {
	var (
		ipIndex     = int32(10)
		service     *corev1.Service
		serviceKey  types.NamespacedName
		serviceSpec corev1.ServiceSpec
		mutateSpec  patch.MutateFn
	)

	BeforeEach(func() {
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("service-%d", rand.Int31()),
				Namespace: "default",
			},
		}

		serviceSpec = corev1.ServiceSpec{
			ClusterIP: fmt.Sprintf("10.0.0.%d", atomic.AddInt32(&ipIndex, 1)),
			Selector:  map[string]string{"foo": "var"},
			Type:      corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		}

		serviceKey = types.NamespacedName{
			Name:      service.Name,
			Namespace: service.Namespace,
		}

		mutateSpec = serviceSpecR(service, serviceSpec)

	})

	It("creates a new object if one doesn't exists", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, mutateSpec)

		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("returning OperationResultCreated")
		Expect(op).To(BeEquivalentTo(patch.OperationResultCreated))

		By("actually having the service created")
		fetched := &corev1.Service{}
		Expect(c.Get(context.TODO(), serviceKey, fetched)).To(Succeed())

		By("being mutated by MutateFn")
		Expect(fetched.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
		Expect(fetched.Spec.Ports).To(HaveLen(1))
		Expect(fetched.Spec.Ports[0].Name).To(Equal("http"))
	})

	It("updates existing object", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, mutateSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(op).To(BeEquivalentTo(patch.OperationResultCreated))

		op, err = patch.CreateOrPatch(context.TODO(), c, service, serviceAnnotation(service, "foo", "bar"))
		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("returning OperationResultUpdated")
		Expect(op).To(BeEquivalentTo(patch.OperationResultUpdated))

		By("actually having the service annotations set")
		fetched := &corev1.Service{}
		Expect(c.Get(context.TODO(), serviceKey, fetched)).To(Succeed())
		Expect(fetched.Annotations).ToNot(BeNil())
		Expect(fetched.Annotations).To(HaveLen(1))
		Expect(fetched.Annotations["foo"]).To(Equal("bar"))
	})

	It("updates only changed objects", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, mutateSpec)

		Expect(op).To(BeEquivalentTo(patch.OperationResultCreated))
		Expect(err).NotTo(HaveOccurred())

		op, err = patch.CreateOrPatch(context.TODO(), c, service, noOpMutate)
		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("returning OperationResultNone")
		Expect(op).To(BeEquivalentTo(patch.OperationResultNone))
	})

	It("updates only changed status", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, mutateSpec)

		Expect(op).To(BeEquivalentTo(patch.OperationResultCreated))
		Expect(err).NotTo(HaveOccurred())

		serviceStatus := corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						Hostname: "hello.world",
					},
				},
			},
		}
		op, err = patch.CreateOrPatch(context.TODO(), c, service, serviceStatusR(service, serviceStatus))
		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("returning OperationResultUpdatedStatusOnly")
		Expect(op).To(BeEquivalentTo(patch.OperationResultUpdatedStatusOnly))
	})

	It("updates resource and status", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, mutateSpec)

		Expect(op).To(BeEquivalentTo(patch.OperationResultCreated))
		Expect(err).NotTo(HaveOccurred())

		serviceSpec.SessionAffinity = corev1.ServiceAffinityClientIP
		serviceStatus := corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						Hostname: "hello.world",
					},
				},
			},
		}
		op, err = patch.CreateOrPatch(context.TODO(), c, service, func() error {
			Expect(serviceStatusR(service, serviceStatus)()).To(Succeed())
			return mutateSpec()
		})
		By("returning no error")
		Expect(err).NotTo(HaveOccurred())

		By("returning OperationResultUpdatedStatus")
		Expect(op).To(BeEquivalentTo(patch.OperationResultUpdatedStatus))
	})

	It("errors when MutateFn changes object name on creation", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, func() error {
			Expect(mutateSpec()).To(Succeed())
			return serviceRenamer(service)()
		})

		By("returning error")
		Expect(err).To(HaveOccurred())

		By("returning OperationResultNone")
		Expect(op).To(BeEquivalentTo(patch.OperationResultNone))
	})

	It("errors when MutateFn renames an object", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, mutateSpec)

		Expect(op).To(BeEquivalentTo(patch.OperationResultCreated))
		Expect(err).NotTo(HaveOccurred())

		op, err = patch.CreateOrPatch(context.TODO(), c, service, serviceRenamer(service))

		By("returning error")
		Expect(err).To(HaveOccurred())

		By("returning OperationResultNone")
		Expect(op).To(BeEquivalentTo(patch.OperationResultNone))
	})

	It("errors when object namespace changes", func() {
		op, err := patch.CreateOrPatch(context.TODO(), c, service, mutateSpec)

		Expect(op).To(BeEquivalentTo(patch.OperationResultCreated))
		Expect(err).NotTo(HaveOccurred())

		op, err = patch.CreateOrPatch(context.TODO(), c, service, serviceNamespaceChanger(service))

		By("returning error")
		Expect(err).To(HaveOccurred())

		By("returning OperationResultNone")
		Expect(op).To(BeEquivalentTo(patch.OperationResultNone))
	})

	It("aborts immediately if there was an error initially retrieving the object", func() {
		op, err := patch.CreateOrPatch(context.TODO(), errorReader{Client: c}, service, func() error {
			Fail("Mutation method should not run")
			return nil
		})

		Expect(op).To(BeEquivalentTo(patch.OperationResultNone))
		Expect(err).To(HaveOccurred())
	})
})

func noOpMutate() error {
	return nil
}

func serviceAnnotation(service *corev1.Service, key, val string) patch.MutateFn {
	return func() error {
		if service.Annotations == nil {
			service.Annotations = map[string]string{}
		}
		service.Annotations[key] = val
		return nil
	}
}

func serviceSpecR(service *corev1.Service, spec corev1.ServiceSpec) patch.MutateFn {
	return func() error {
		service.Spec = spec
		return nil
	}
}

func serviceStatusR(service *corev1.Service, status corev1.ServiceStatus) patch.MutateFn {
	return func() error {
		service.Status = status
		return nil
	}
}

func serviceRenamer(service *corev1.Service) patch.MutateFn {
	return func() error {
		service.Name = fmt.Sprintf("%s-1", service.Name)
		return nil
	}
}

func serviceNamespaceChanger(service *corev1.Service) patch.MutateFn {
	return func() error {
		service.Namespace = fmt.Sprintf("%s-1", service.Namespace)
		return nil

	}
}

type errorReader struct {
	client.Client
}

func (e errorReader) Get(ctx context.Context, key client.ObjectKey, into runtime.Object) error {
	return fmt.Errorf("unexpected error")
}
