package main

import (
	"context"
	"github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	"github.com/myid/myresource-crd/pkg/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateCR(ctx context.Context, clientset *fake.Clientset, name, namespace, image, memory string) (cr *v1alpha1.MyResource, err error) {
	crToCreate := &v1alpha1.MyResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "mygroup.example.com/v1alpha1",
			Kind:       "MyResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.MyResourceSpec{
			Image:  image,
			Memory: resource.MustParse(memory),
		},
	}
	return clientset.MygroupV1alpha1().MyResources(namespace).Create(ctx, crToCreate, metav1.CreateOptions{})
}
