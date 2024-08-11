package main

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/pointer"
)

func main() {
	spec := appsv1.DeploymentSpec{
		Replicas: pointer.Int32(3),
	}
	_ = spec

	// de-ref, given a pointer & a default val
	replicas := pointer.Int32Deref(spec.Replicas, 1)
	fmt.Println(replicas)

	// compare two referenced val
	spec1 := appsv1.DeploymentSpec{
		Replicas: pointer.Int32(3),
	}
	spec2 := appsv1.DeploymentSpec{
		Replicas: pointer.Int32(2),
	}
	isEqual := pointer.Int32Equal(spec1.Replicas, spec2.Replicas)
	fmt.Println(isEqual)

}
