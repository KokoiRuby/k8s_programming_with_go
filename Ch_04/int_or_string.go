package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func main() {
	ios1 := intstr.FromInt32(1)
	fmt.Printf("ios1: %v\n", ios1.IntValue())

	ios2 := intstr.FromString("1")
	fmt.Printf("ios1: %s\n", ios2.String())

	ios3 := intstr.Parse("100")
	fmt.Printf("ios3 as string: %s\n", ios3.String())
	fmt.Printf("ios3 as int: %d\n", ios3.IntValue())

	ios4 := intstr.Parse("value")
	fmt.Printf("ios4 as string: %s\n", ios4.String())
	fmt.Printf("ios4as int: %d\n", ios4.IntValue())

	fmt.Printf("ios4 or 'default': %s\n", intstr.ValueOrDefault(&ios4, intstr.Parse("default")))
	fmt.Printf("nil or 'default': %s\n", intstr.ValueOrDefault(nil, intstr.Parse("default")))

	ios5 := intstr.Parse("10%")
	scaled, _ := intstr.GetScaledValueFromIntOrPercent(&ios5, 5000, true)
	fmt.Printf("10%% of 5000: %d\n", scaled)
}
