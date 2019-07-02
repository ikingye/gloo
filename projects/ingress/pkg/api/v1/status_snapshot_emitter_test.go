// Code generated by solo-kit. DO NOT EDIT.

// +build solokit

package v1

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/kubeutils"
	"github.com/solo-io/go-utils/log"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/memory"
	"github.com/solo-io/solo-kit/test/helpers"
	"k8s.io/client-go/kubernetes"

	// Needed to run tests in GKE
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	// From https://github.com/kubernetes/client-go/blob/53c7adfd0294caa142d961e1f780f74081d5b15f/examples/out-of-cluster-client-configuration/main.go#L31
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var _ = Describe("V1Emitter", func() {
	if os.Getenv("RUN_KUBE_TESTS") != "1" {
		log.Printf("This test creates kubernetes resources and is disabled by default. To enable, set RUN_KUBE_TESTS=1 in your env.")
		return
	}
	var (
		namespace1        string
		namespace2        string
		name1, name2      = "angela" + helpers.RandString(3), "bob" + helpers.RandString(3)
		kube              kubernetes.Interface
		emitter           StatusEmitter
		kubeServiceClient KubeServiceClient
		ingressClient     IngressClient
	)

	BeforeEach(func() {
		namespace1 = helpers.RandString(8)
		namespace2 = helpers.RandString(8)
		kube = helpers.MustKubeClient()
		err := kubeutils.CreateNamespacesInParallel(kube, namespace1, namespace2)
		Expect(err).NotTo(HaveOccurred())
		// KubeService Constructor
		kubeServiceClientFactory := &factory.MemoryResourceClientFactory{
			Cache: memory.NewInMemoryResourceCache(),
		}

		kubeServiceClient, err = NewKubeServiceClient(kubeServiceClientFactory)
		Expect(err).NotTo(HaveOccurred())
		// Ingress Constructor
		ingressClientFactory := &factory.MemoryResourceClientFactory{
			Cache: memory.NewInMemoryResourceCache(),
		}

		ingressClient, err = NewIngressClient(ingressClientFactory)
		Expect(err).NotTo(HaveOccurred())
		emitter = NewStatusEmitter(kubeServiceClient, ingressClient)
	})
	AfterEach(func() {
		err := kubeutils.DeleteNamespacesInParallelBlocking(kube, namespace1, namespace2)
		Expect(err).NotTo(HaveOccurred())
	})
	It("tracks snapshots on changes to any resource", func() {
		ctx := context.Background()
		err := emitter.Register()
		Expect(err).NotTo(HaveOccurred())

		snapshots, errs, err := emitter.Snapshots([]string{namespace1, namespace2}, clients.WatchOpts{
			Ctx:         ctx,
			RefreshRate: time.Second,
		})
		Expect(err).NotTo(HaveOccurred())

		var snap *StatusSnapshot

		/*
			KubeService
		*/

		assertSnapshotServices := func(expectServices KubeServiceList, unexpectServices KubeServiceList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectServices {
						if _, err := snap.Services.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectServices {
						if _, err := snap.Services.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := kubeServiceClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := kubeServiceClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		kubeService1a, err := kubeServiceClient.Write(NewKubeService(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		kubeService1b, err := kubeServiceClient.Write(NewKubeService(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(KubeServiceList{kubeService1a, kubeService1b}, nil)
		kubeService2a, err := kubeServiceClient.Write(NewKubeService(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		kubeService2b, err := kubeServiceClient.Write(NewKubeService(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(KubeServiceList{kubeService1a, kubeService1b, kubeService2a, kubeService2b}, nil)

		err = kubeServiceClient.Delete(kubeService2a.GetMetadata().Namespace, kubeService2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = kubeServiceClient.Delete(kubeService2b.GetMetadata().Namespace, kubeService2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(KubeServiceList{kubeService1a, kubeService1b}, KubeServiceList{kubeService2a, kubeService2b})

		err = kubeServiceClient.Delete(kubeService1a.GetMetadata().Namespace, kubeService1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = kubeServiceClient.Delete(kubeService1b.GetMetadata().Namespace, kubeService1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(nil, KubeServiceList{kubeService1a, kubeService1b, kubeService2a, kubeService2b})

		/*
			Ingress
		*/

		assertSnapshotIngresses := func(expectIngresses IngressList, unexpectIngresses IngressList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectIngresses {
						if _, err := snap.Ingresses.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectIngresses {
						if _, err := snap.Ingresses.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := ingressClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := ingressClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		ingress1a, err := ingressClient.Write(NewIngress(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		ingress1b, err := ingressClient.Write(NewIngress(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(IngressList{ingress1a, ingress1b}, nil)
		ingress2a, err := ingressClient.Write(NewIngress(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		ingress2b, err := ingressClient.Write(NewIngress(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(IngressList{ingress1a, ingress1b, ingress2a, ingress2b}, nil)

		err = ingressClient.Delete(ingress2a.GetMetadata().Namespace, ingress2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = ingressClient.Delete(ingress2b.GetMetadata().Namespace, ingress2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(IngressList{ingress1a, ingress1b}, IngressList{ingress2a, ingress2b})

		err = ingressClient.Delete(ingress1a.GetMetadata().Namespace, ingress1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = ingressClient.Delete(ingress1b.GetMetadata().Namespace, ingress1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(nil, IngressList{ingress1a, ingress1b, ingress2a, ingress2b})
	})
	It("tracks snapshots on changes to any resource using AllNamespace", func() {
		ctx := context.Background()
		err := emitter.Register()
		Expect(err).NotTo(HaveOccurred())

		snapshots, errs, err := emitter.Snapshots([]string{""}, clients.WatchOpts{
			Ctx:         ctx,
			RefreshRate: time.Second,
		})
		Expect(err).NotTo(HaveOccurred())

		var snap *StatusSnapshot

		/*
			KubeService
		*/

		assertSnapshotServices := func(expectServices KubeServiceList, unexpectServices KubeServiceList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectServices {
						if _, err := snap.Services.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectServices {
						if _, err := snap.Services.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := kubeServiceClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := kubeServiceClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		kubeService1a, err := kubeServiceClient.Write(NewKubeService(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		kubeService1b, err := kubeServiceClient.Write(NewKubeService(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(KubeServiceList{kubeService1a, kubeService1b}, nil)
		kubeService2a, err := kubeServiceClient.Write(NewKubeService(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		kubeService2b, err := kubeServiceClient.Write(NewKubeService(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(KubeServiceList{kubeService1a, kubeService1b, kubeService2a, kubeService2b}, nil)

		err = kubeServiceClient.Delete(kubeService2a.GetMetadata().Namespace, kubeService2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = kubeServiceClient.Delete(kubeService2b.GetMetadata().Namespace, kubeService2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(KubeServiceList{kubeService1a, kubeService1b}, KubeServiceList{kubeService2a, kubeService2b})

		err = kubeServiceClient.Delete(kubeService1a.GetMetadata().Namespace, kubeService1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = kubeServiceClient.Delete(kubeService1b.GetMetadata().Namespace, kubeService1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotServices(nil, KubeServiceList{kubeService1a, kubeService1b, kubeService2a, kubeService2b})

		/*
			Ingress
		*/

		assertSnapshotIngresses := func(expectIngresses IngressList, unexpectIngresses IngressList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectIngresses {
						if _, err := snap.Ingresses.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectIngresses {
						if _, err := snap.Ingresses.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := ingressClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := ingressClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		ingress1a, err := ingressClient.Write(NewIngress(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		ingress1b, err := ingressClient.Write(NewIngress(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(IngressList{ingress1a, ingress1b}, nil)
		ingress2a, err := ingressClient.Write(NewIngress(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		ingress2b, err := ingressClient.Write(NewIngress(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(IngressList{ingress1a, ingress1b, ingress2a, ingress2b}, nil)

		err = ingressClient.Delete(ingress2a.GetMetadata().Namespace, ingress2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = ingressClient.Delete(ingress2b.GetMetadata().Namespace, ingress2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(IngressList{ingress1a, ingress1b}, IngressList{ingress2a, ingress2b})

		err = ingressClient.Delete(ingress1a.GetMetadata().Namespace, ingress1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = ingressClient.Delete(ingress1b.GetMetadata().Namespace, ingress1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotIngresses(nil, IngressList{ingress1a, ingress1b, ingress2a, ingress2b})
	})
})
