package main

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func main() {
	// init a Deployment struct
	myDep := appsv1.Deployment{}
	fmt.Printf("%+v\n", myDep)

	myCM := corev1.ConfigMap{}
	myCM.SetName("myConfigMap")
	fmt.Printf("%+v\n", myCM.Name)

	// construct label by go built-in map
	myLabel1 := map[string]string{
		"app.kubernetes.io/component": "my-component",
		"app.kubernetes.io/name":      "my-app",
	}
	fmt.Printf("%+v\n", myLabel1)

	// construct label by apimachinery
	myLabel2 := labels.Set{
		"app.kubernetes.io/component": "my-component",
		"app.kubernetes.io/name":      "my-app",
	}
	fmt.Printf("%+v\n", myLabel2)
}
