package types

type DiningMethod string

const (
	DiningMethodNone    DiningMethod = "none"     // 无
	DiningMethodDineIn  DiningMethod = "dine_in"  // 堂食
	DiningMethodTakeOut DiningMethod = "take_out" // 外带
)

func (d DiningMethod) String() string {
	return string(d)
}
