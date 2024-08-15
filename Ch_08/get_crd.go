package main

import (
	"context"
	"fmt"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := getConfigOnDisk()
	if err != nil {
		panic(err)
	}
	cs, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	ctx := context.TODO()
	crdList, err := cs.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, crd := range crdList.Items {
		fmt.Println(crd.GetName())
	}
}

func getConfigOnDisk() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", "./config")
}
