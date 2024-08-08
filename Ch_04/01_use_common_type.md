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

对应的解引用：接收一个类型引用与该类型的默认值。

```go
func IntDeref(ptr *int, def int) int {
	if ptr != nil {
		return *ptr
	}
	return def
}

replicas := pointer.IntDeref(spec.Replicas, 1)
```

（同类型）指针（对应的值）比较

```go
func IntEqual(a, b *int) bool {
	if (a == nil) != (b == nil) {
		return false
    }
	if a == nil {
		return true
	}
	return *a == *b
}

isEqual := pointer.IntEqual(spec1.Replicas,	spec2.Replicas)
```

## Quantities

定点数值，表示分配的资源量 cpu/mem，最小值：10<sup>-9</sup>

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

表示：`Ki, Mi, ...` (1024) & `K, M, ...` (1000)

inf.Dec

```go
// construtor
func NewDecimalQuantity(b inf.Dec, format Format) *Quantity

// 强制将一个之前通过解析字符串或使用新函数初始化的 Quantity 存储为 inf.Dec。
func (q *Quantity) ToDec() *Quantity

// 获取 Quantity 的 inf.Dec 表示，而不修改其内部表示。
func (q *Quantity) AsDec() *inf.Dec
```

Scaled Integer 将整数表示为实际值的缩放整数 → 3.14 with factor 100 → 314 避免浮点计算中的精度损失。

```go
func NewScaledQuantity(value int64, scale Scale) *Quantity
func (q *Quantity) SetScaled(value int64, scale Scale)
func (q *Quantity) ScaledValue(scale Scale) int64

func NewQuantity(value int64, format Format) *Quantity
func (q *Quantity) Set(value int64)
func (q *Quantity) Value() int64

func NewMilliQuantity(value int64, format Format)
func (q *Quantity) SetMilli(value int64)
func (q *Quantity) MilliValue() int64
```

Operations

```go
func (q *Quantity) Add(y Quantity)
func (q *Quantity) Sub(y Quantity)
func (q *Quantity) Cmp(y Quantity)
func (q *Quantity) CmpInt64(y int64)
func (q *Quantity) Neg()
func (q Quantity) Equal(v Quantity) bool
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
func FromInt(val int) IntOrString
func FromString(val string) IntOrString
func Parse(val string) IntOrString

func (intstr *IntOrString) String() string
func (intstr *IntOrString) IntValue() int

func ValueOrDefault(intOrPercent *IntOrString, defaultValue IntOrString) *IntOrString
func GetScaledValueFromIntOrPercent(intOrPercent *IntOrString, total int, roundUp bool) (int, error)
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

