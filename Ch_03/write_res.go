package main

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
)

func main() {
	// # Specific Content in core/v1
	// ## ResourceList - requests & cpu
	requests := corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewQuantity(250, resource.BinarySI),
		corev1.ResourceMemory: *resource.NewQuantity(64*1024*1024, resource.BinarySI),
	}
	_ = requests
	limits := corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewQuantity(500, resource.BinarySI),
		corev1.ResourceMemory: *resource.NewQuantity(128*1024*1024, resource.BinarySI),
	}
	_ = limits

	// # Writing Kubernetes Resources in Go
	// ## Importing the package
	myDep := appsv1.Deployment{}
	fmt.Printf("%+v\n", myDep)

	// ## The ObjectMeta fields
	// ### Name
	myCM := corev1.ConfigMap{}
	myCM.SetName("myConfigMap")
	fmt.Printf("%+v\n", myCM.Name)

	// ### Labels and annotations
	// construct label by go built-in map
	myLabel1 := map[string]string{
		"app.kubernetes.io/component": "my-component",
		"app.kubernetes.io/name":      "my-app",
	}
	fmt.Printf("%+v\n", myLabel1)

	// add label
	myLabel1["myKey"] = "myValue1"

	// construct label by apimachinery
	myLabel2 := labels.Set{
		"app.kubernetes.io/component": "my-component",
		"app.kubernetes.io/name":      "my-app",
	}
	fmt.Printf("%+v\n", myLabel2)

	// ### OwnerReferences
	// get the obj to reference
	// get client set
	clientSet, err := getClientSet()
	if err != nil {
		panic(err)
	}

	// get pod
	pod, err := clientSet.CoreV1().Pods("myns").Get(context.TODO(), "mypodname", metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	// Solution 1: set the APIVersion and Kind of the Pod then copy all info from the pod
	pod.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))
	ownerRef := metav1.OwnerReference{
		APIVersion: pod.APIVersion,
		Kind:       pod.Kind,
		Name:       pod.GetName(),
		UID:        pod.GetUID(),
	}
	_ = ownerRef

	// Solution 2: Copy name and uid from pod then set APIVersion and Kind on the OwnerReference
	ownerRef = metav1.OwnerReference{
		Name: pod.GetName(),
		UID:  pod.GetUID(),
	}
	ownerRef.APIVersion, ownerRef.Kind = corev1.SchemeGroupVersion.WithKind("Pod").ToAPIVersionAndKind()

	// #### Setting Controller, must be a pointer
	// Solution 1: declare a value and use its address
	controller := true
	ownerRef.Controller = &controller

	// ## Comparison with writing YAML manifests
	// Solution 2: use the BoolPtr function
	ownerRef.Controller = pointer.BoolPtr(controller)

	pod1 := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "runtime",
					Image: "nginx",
				},
			},
		},
	}
	pod1.SetName("my-pod")
	pod1.SetLabels(map[string]string{
		"component": "my-component",
	})

	// Solution 2
	pod2 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
			Labels: map[string]string{
				"component": "myComponent",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "runtime",
					Image: "nginx",
				},
			},
		},
	}

	_ = pod2
}

func getClientSet() (*kubernetes.Clientset, error) {
	config, err :=
		clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			nil,
		).ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
