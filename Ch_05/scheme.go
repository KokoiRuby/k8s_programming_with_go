package main

import (
	"bytes"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
)

func main() {
	// init & reg API = kind/type with group & version
	scheme1 := runtime.NewScheme()
	scheme1.AddKnownTypes(schema.GroupVersion{
		Group:   "",
		Version: "v1",
	},
		&corev1.Pod{},
		&corev1.ConfigMap{})

	// versioned
	scheme2 := runtime.NewScheme()
	scheme2.AddKnownTypes(schema.GroupVersion{
		Group:   "apps",
		Version: "v1",
	}, &appsv1.Deployment{})
	scheme2.AddKnownTypes(schema.GroupVersion{
		Group:   "apps",
		Version: "v1beta1",
	}, &appsv1beta1.Deployment{})

	// mapping GVK & Types
	// get kind/type given GV
	types := scheme2.KnownTypes(schema.GroupVersion{
		Group:   "apps",
		Version: "v1",
	})
	fmt.Println(types)

	// get ver given GK
	groupVer := scheme2.VersionsForGroupKind(schema.GroupKind{
		Group: "apps",
		Kind:  "Deployment",
	})
	fmt.Println(groupVer)

	// get gvk given an obj
	gvk, _, _ := scheme2.ObjectKinds(&appsv1.Deployment{})
	fmt.Println(gvk)

	// build an obj given gvk
	obj, _ := scheme2.New(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})
	fmt.Println(obj)

	// Conversion - btw kinds of the same groups
	// add conversion func - btw  appsv1 & appsv1beta
	err := scheme2.AddConversionFunc((*appsv1.Deployment)(nil), (*appsv1.Deployment)(nil), func(a, b interface{}, v conversion.Scope) error {
		v1 := a.(*appsv1.Deployment)
		v1beta := b.(*appsv1beta1.Deployment)
		// Conversion
		_, _ = v1, v1beta
		return nil
	})
	if err != nil {
		return
	}

	// convert by func
	v1Deploy := appsv1.Deployment{}
	v1Deploy.SetName("myname")
	v1Deploy.APIVersion, v1Deploy.Kind = appsv1.SchemeGroupVersion.WithKind("Deployment").ToAPIVersionAndKind()
	fmt.Println(v1Deploy)

	var v1beta1Deployment appsv1beta1.Deployment
	err = scheme2.Convert(&v1Deploy, &v1beta1Deployment, nil)

	// Serialization
	jsonSerializer := jsonserializer.NewSerializerWithOptions(
		jsonserializer.DefaultMetaFactory, scheme2, scheme2, jsonserializer.SerializerOptions{
			Yaml:   false,
			Pretty: true,  // or false for one-line JSON
			Strict: false, // or true to check duplicates
		})

	protoBufSerializer := protobuf.NewSerializer(scheme2, scheme2)
	_ = protoBufSerializer

	// obj → JSON
	var buffer bytes.Buffer
	err = jsonSerializer.Encode(&v1Deploy, &buffer)
	if err != nil {
		return
	}
	fmt.Println(buffer.String())

	// JSON → obj
	var decodedDeployment appsv1.Deployment
	json := `{"kind": "Deployment", "apiVersion": "apps/v1", "metadata":{"name":"myname"}}`
	_, groupVersionKind, _ := jsonSerializer.Decode([]byte(json), nil, &decodedDeployment)
	fmt.Printf("obj: %v\ngvk: %s\n", decodedDeployment, groupVersionKind)
}
