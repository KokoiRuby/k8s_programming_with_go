package main

import (
	"context"
	myresourcev1alpha1 "github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func CreateMyResource(dynamicClient dynamic.Interface, u *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvr := myresourcev1alpha1.SchemeGroupVersion.WithResource("myresource")
	return dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), u, v1.CreateOptions{})
}

func getResource() (*unstructured.Unstructured, error) {
	myres := unstructured.Unstructured{}
	myres.SetGroupVersionKind(
		myresourcev1alpha1.SchemeGroupVersion.
			WithKind("MyResource"))
	myres.SetName("myres1")
	myres.SetNamespace("default")
	// spec
	err := unstructured.SetNestedField(
		myres.Object,
		"nginx",
		"spec", "image",
	)
	if err != nil {
		return nil, err
	}
	// Use int64
	err = unstructured.SetNestedField(
		myres.Object,
		int64(1024*1024*1024),
		"spec", "memory",
	)
	if err != nil {
		return nil, err
	}
	// or use string
	err = unstructured.SetNestedField(
		myres.Object,
		"1024Mi",
		"spec", "memory",
	)
	if err != nil {
		return nil, err
	}
	return &myres, nil
}
