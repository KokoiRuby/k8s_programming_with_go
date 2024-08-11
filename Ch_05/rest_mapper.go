package main

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	scheme := runtime.NewScheme()

	// add known types to scheme
	scheme.AddKnownTypes(schema.GroupVersion{
		Group:   "apps",
		Version: "v1",
	}, &appsv1.Deployment{})
	scheme.AddKnownTypes(schema.GroupVersion{
		Group:   "apps",
		Version: "v1beta1",
	}, &appsv1beta1.Deployment{})

	// get version given group & kind
	groupVer := scheme.VersionsForGroupKind(schema.GroupKind{
		Group: "apps",
		Kind:  "Deployment",
	})
	fmt.Println(groupVer)

	// add mapping into rest mapper
	restMapper := meta.NewDefaultRESTMapper(groupVer)
	restMapper.Add(appsv1.SchemeGroupVersion.WithKind("Deployment"), nil)
	restMapper.Add(appsv1beta1.SchemeGroupVersion.WithKind("Deployment"), nil)

	// RESTMapping: {
	//    Resource
	//    GroupVersionKind
	//    Scope
	// }
	mapping, _ := restMapper.RESTMapping(schema.GroupKind{
		Group: "apps",
		Kind:  "Deployment",
	})
	fmt.Printf("single mapping: %+v\n", *mapping)

	mappings, _ := restMapper.RESTMappings(schema.GroupKind{
		Group: "apps",
		Kind:  "Deployment",
	})
	for _, m := range mappings {
		fmt.Printf("mapping: %+v\n", m)
	}

	// GVK
	kinds, _ := restMapper.KindsFor(schema.GroupVersionResource{
		Group:    "",
		Version:  "",
		Resource: "deployment",
	})
	fmt.Printf("kinds: %+v\n", kinds)

	// GVR
	resources, _ := restMapper.ResourcesFor(schema.GroupVersionResource{
		Group:    "",
		Version:  "",
		Resource: "deployment",
	})
	fmt.Printf("resources: %+v\n", resources)

}
