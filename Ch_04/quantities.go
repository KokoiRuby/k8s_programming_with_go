package main

import (
	"fmt"
	"gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"
)

func main() {
	// parse string as quantity
	q1 := resource.MustParse("1Mi")
	q1, err := resource.ParseQuantity("1Mi")
	_, _ = q1, err

	// use inf.Dec as quantity
	newDec := inf.NewDec(4, 3)
	fmt.Printf("newDec: %s\n", newDec)
	q2 := resource.NewDecimalQuantity(*newDec, resource.DecimalExponent)
	fmt.Printf("q2: %s\n", q2)
	toDec := q2.ToDec()
	fmt.Printf("toDec: %s\n", toDec)
	asDec := q2.AsDec()
	fmt.Printf("asDec: %s\n", asDec)

	// use scaled integer as quantity
	q3 := resource.NewScaledQuantity(4, 3)
	fmt.Printf("q3: %s\n", q3)
	q3.SetScaled(5, 6)
	fmt.Printf("q3: %s\n", q3)
	fmt.Printf("q3 scaled to 3: %d\n", q3.ScaledValue(3))
	fmt.Printf("q3 scaled to 0: %d\n", q3.ScaledValue(0))

	q4 := resource.NewQuantity(4000, resource.DecimalSI)
	fmt.Printf("q4: %s\n", q4)
	q5 := resource.NewQuantity(1024, resource.BinarySI)
	fmt.Printf("q5: %s\n", q5)
	q6 := resource.NewQuantity(4000, resource.DecimalExponent)
	fmt.Printf("q6: %s\n", q6)

	q8 := resource.NewMilliQuantity(5, resource.DecimalExponent)
	fmt.Printf("q8: %s\n", q8)
	q8.SetMilli(6)
	fmt.Printf("q8: %s\n", q8)

	fmt.Printf("milli value of q8: %d\n", q8.MilliValue())

	// quantity op
	q9 := resource.MustParse("4M")
	q10 := resource.MustParse("3M")
	q9.Add(q10)
	fmt.Printf("q9: %s\n", q9.String())
	q9.Sub(q10)
	fmt.Printf("q9: %s\n", q9.String())
	cmp := q9.Cmp(q10)
	fmt.Printf("4M >? 3M: %d\n", cmp)
	cmp = q9.CmpInt64(4_000_000)
	fmt.Printf("4M >? 4.000.000: %d\n", cmp)
	q9.Neg()
	fmt.Printf("negative of 4M: %s\n", q9.String())
	eq := q9.Equal(q10)
	fmt.Printf("4M ==? 3M: %v\n", eq)
}
