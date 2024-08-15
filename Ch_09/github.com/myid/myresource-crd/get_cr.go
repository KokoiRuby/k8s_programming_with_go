package main

import (
	"context"
	"fmt"
	"github.com/myid/myresource-crd/pkg/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := getConfigOnDisk()
	if err != nil {
		panic(err)
	}
	cs, err := clientset.NewForConfig(config) // clientset for cr by client-gen
	if err != nil {
		panic(err)
	}
	ctx := context.TODO()
	crList, err := cs.MygroupV1alpha1().MyResources("default").List(ctx, v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, cr := range crList.Items {
		fmt.Println(cr.Name)
	}
}

func getConfigOnDisk() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", "./config")
}
