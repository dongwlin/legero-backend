package types

type Noodle string

const (
	NoodleNone     Noodle = "none"      // 无
	NoodleFlatRice Noodle = "flat_rice" // 河粉
	NoodleThinRice Noodle = "thin_rice" // 米粉
	NoodleYi       Noodle = "yi"        // 伊面
	NoodleCustom   Noodle = "custom"    // 自定义
)

func (n Noodle) String() string {
	return string(n)
}

type Size string

const (
	SizeNone   Size = "none"   // 无
	SizeSmall  Size = "small"  // 小
	SizeMiddle Size = "middle" // 中
	SizeLarge  Size = "large"  // 大
	SizeCustom Size = "custom" // 自定义
)

func (s Size) String() string {
	return string(s)
}
