package types

type Adjustment string

const (
	AdjustmentNone    Adjustment = "none"    // 无
	AdjustmentRegular Adjustment = "regular" // 正常
	AdjustmentLight   Adjustment = "light"   // 少
	AdjustmentHeavy   Adjustment = "heavy"   // 多
	AdjustmentExclude Adjustment = "exclude" // 不要
)

func (a Adjustment) String() string {
	return string(a)
}
