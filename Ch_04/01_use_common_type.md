## Pointers

Go 中结构体中的**可选值 Optional values** 通常被声明一个指向某个值的指针。无可选值为 `nil`。

```go
import (
	"k8s.io/utils/pointer"
)
```

`Int[32|64]`, `Bool`, `String`, `Float[32|64]`, `Duration` 接收一个类型的值，返回类型指针指向该值。

```go
func Int(i int) *int {
	return &i
}

spec := appsv1.DeploymentSpec{ Replicas: pointer.Int32(3), [...]}
```

对应的解引用：接收一个类型引用与该类型的**默认值**。

```go
replicas := pointer.IntDeref(spec.Replicas, 1)
```

（同类型）指针（对应的值）比较

```go
spec1 := appsv1.DeploymentSpec{
		Replicas: pointer.Int32(3),
	}
	spec2 := appsv1.DeploymentSpec{
		Replicas: pointer.Int32(2),
	}
	isEqual := pointer.Int32Equal(spec1.Replicas, spec2.Replicas)
	fmt.Println(isEqual)
```

## Quantities

定点数值，表示分配的资源量 cpu/mem，最小值：10<sup>-9</sup>

表示：`Ki, Mi, ...` (1024) & `K, M, ...` (1000)

```go
import (
	"k8s.io/apimachinery/pkg/api/resource"
)
```

解析字符串

```go
// 解析，panic if 无法表示
func MustParse(str string) Quantity

// 解析，error if 无法表示
func ParseQuantity(str string) (Quantity, error)
```

```go
q1 := resource.MustParse("1Mi")
q1, err := resource.ParseQuantity("1Mi")
```

inf.Dec

```go
// construtor
func NewDecimalQuantity(b inf.Dec, format Format) *Quantity

// promotes the quantity in place to use an inf.Dec representation and returns itself.
func (q *Quantity) ToDec() *Quantity

// returns the quantity as represented by a scaled inf.Dec.
func (q *Quantity) AsDec() *inf.Dec
```

```go
newDec := inf.NewDec(4, 3)  
q2 := resource.NewDecimalQuantity(*newDec, resource.DecimalExponent)
toDec := q2.ToDec()
asDec := q2.AsDec()
// output:
// 0.004
// 4e-3
// 4e-3
// 0.004
```

Scaled Integer 将整数表示为实际值的缩放整数 → 3.14 with factor 100 → 314 避免浮点计算中的精度损失。

```go
// declares a Quantity by giving an int64 value and a scale.
func NewScaledQuantity(value int64, scale Scale) *Quantity
// overrides a Quantity value with a scaled integer
func (q *Quantity) SetScaled(value int64, scale Scale)
// gets a representation of the Quantity as an integer without modifying the internal repr
func (q *Quantity) ScaledValue(scale Scale) int64

// declares a Quantity by giving an int64 value, the scale being fixed to 0
func NewQuantity(value int64, format Format) *Quantity
// overrides a Quantity value with an integer
func (q *Quantity) Set(value int64)
// gets a repr of the Quantity as an integer, without modifying the internal repr
func (q *Quantity) Value() int64

// declares a Quantity by giving an int64 value, the scale being fixed to −3
func NewMilliQuantity(value int64, format Format)
// overrides a Quantity value with an integer
func (q *Quantity) SetMilli(value int64)
// gets a repr of the Quantity as an integer, without modifying the internal repr
func (q *Quantity) MilliValue() int64
```

```go
q3 := resource.NewScaledQuantity(4, 3)  // 4K
q3.SetScaled(5, 6)                      // 5M
fmt.Printf("q3 scaled to 3: %d\n", q3.ScaledValue(3))  // 5000
fmt.Printf("q3 scaled to 0: %d\n", q3.ScaledValue(0))  // 5000000

q4 := resource.NewQuantity(4000, resource.DecimalSI)       // 4K
q5 := resource.NewQuantity(1024, resource.BinarySI)        // 1Ki
q6 := resource.NewQuantity(4000, resource.DecimalExponent) // 4e3

q8 := resource.NewMilliQuantity(5, resource.DecimalExponent) // 5e-3
q8.SetMilli(6)                                               // 6e-3
fmt.Printf("milli value of q8: %d\n", q8.MilliValue())       // 6
```

Operations

```go
func (q *Quantity) Add(y Quantity)
func (q *Quantity) Sub(y Quantity)
func (q *Quantity) Cmp(y Quantity)
// 0 equal; 1 greater, -1 less
func (q *Quantity) CmpInt64(y int64)
func (q *Quantity) Neg()
func (q Quantity) Equal(v Quantity) bool
```

```go
q9 := resource.MustParse("4M")
q10 := resource.MustParse("3M")
q9.Add(q10)                  // 7M
q9.Sub(q10)                  // 4
cmp := q9.Cmp(q10)           // 1 
cmp = q9.CmpInt64(4_000_000) // 0
q9.Neg()                     // -4M
eq := q9.Equal(q10)          // false
```

## IntOrStrings

K8s 有些字段比如 port 即可以接收 interger 也可以接收 string。

```go
import (
	"k8s.io/apimachinery/pkg/util/intstr"
)

type IntOrString struct {
	Type Type     // either Int or String
	IntVal int32
	StrVal string
}
```

```go
// constrcutor
func FromInt32(val int) IntOrString
func FromString(val string) IntOrString
func Parse(val string) IntOrString

func (intstr *IntOrString) String() string
// 0 if parsing (string to int) fails
func (intstr *IntOrString) IntValue() int

// returns the intOrPercent value if not nil, or the defaultValue.
func ValueOrDefault(intOrPercent *IntOrString, defaultValue IntOrString) *IntOrString
// expected to contain either an integer or a percentage. If the value is an integer, it is returned as is.
func GetScaledValueFromIntOrPercent(intOrPercent *IntOrString, total int, roundUp bool) (int, error)
```

```go
ios1 := intstr.FromInt32(1)
ios1.IntValue() // 1

ios2 := intstr.FromString("1")
ios2.String()   // 1

ios3 := intstr.Parse("100")
ios3.String()   // 100
ios3.IntValue() // 100

ios4 := intstr.Parse("value")
ios4.String()   // value
ios4.IntValue() // 0
intstr.ValueOrDefault(&ios4, intstr.Parse("default")) // value
intstr.ValueOrDefault(nil, intstr.Parse("default"))   // default

ios5 := intstr.Parse("10%")
ntstr.GetScaledValueFromIntOrPercent(&ios5, 5000, true)  // 500
```

## Time

`metav1.Time` as Wrapper of Go `time.Time`

```go
import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
```

Factory

```go
func NewTime(time time.Time) Time
func Date(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) Time
func Now() Time
func Unix(sec int64, nsec int64) Time
```

Operations

```go
func (t *Time) DeepCopyInto(out *Time)
func (t *Time) IsZero() bool
func (t *Time) Before(u *Time) bool
func (t *Time) Equal(u *Time) bool
func (t Time) Rfc3339Copy() Time
```

