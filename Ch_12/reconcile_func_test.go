package main

import (
	"fmt"
	mygroupv1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//var k8sClient client.Client
//var ctx = context.Background()

var _ = Describe("MyResource controller", func() {
	When("When creating a MyResource instance", func() {
		var (
			name       string
			image      string
			namespace  = "default"
			myRes      mygroupv1alpha1.MyResource
			ownerRef   *metav1.OwnerReference
			deployName string
		)

		BeforeEach(func() {
			name = fmt.Sprintf("myres-%d", rand.Intn(1000))
			image = fmt.Sprintf("myimage-%d", rand.Intn(1000))
			myRes = mygroupv1alpha1.MyResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: mygroupv1alpha1.MyResourceSpec{
					Image: image,
				},
			}
			err := k8sClient.Create(ctx, &myRes)
			Expect(err).NotTo(HaveOccurred())
			ownerRef = metav1.NewControllerRef(&myRes, mygroupv1alpha1.SchemeGroupVersion.WithKind("MyResource"))
			deployName = fmt.Sprintf("%s-deployment", name)
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, &myRes)
			if err != nil {
				return
			}
		})

		It("should create a deployment", func() {
			var deploy appsv1.Deployment
			// timeout + polling interval
			Eventually(deploymentExists(deployName, namespace, &deploy), 10, 1).Should(BeTrue())
		})

		When("deployment is found", func() {
			var deploy appsv1.Deployment
			BeforeEach(func() {
				Eventually(deploymentExists(deployName, namespace, &deploy), 10, 1).Should(BeTrue())
			})

			It("should be owned by the MyResource instance", func() {
				Expect(deploy.GetOwnerReferences()).To(ContainElement(*ownerRef))
			})

			It("should use the image specified in MyResource instance", func() {
				Expect(deploy.Spec.Template.Spec.Containers[0].Image).To(Equal(image))
			})

			When("deployment ReadyReplicas is 1", func() {
				BeforeEach(func() {
					deploy.Status.Replicas = 1
					deploy.Status.ReadyReplicas = 1
					err := k8sClient.Status().Update(ctx, &deploy)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should set status ready for MyReource instance", func() {
					Eventually(getMyResourceState(deployName, namespace, &deploy), 10, 1).Should(BeTrue())
				})
			})
		})

	})
})

func deploymentExists(name, namespace string, deploy *appsv1.Deployment) func() bool {
	return func() bool {
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, deploy)
		return err == nil
	}
}

func getMyResourceState(name, namespace string, deploy *appsv1.Deployment) func() (string, error) {
	return func() (string, error) {
		myRes := &mygroupv1alpha1.MyResource{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, myRes)
		if err != nil {
			return "", err
		}
		return myRes.Status.State, nil
	}
}
