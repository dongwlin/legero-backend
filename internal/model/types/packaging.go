package types

type Packaging string

const (
	PackagingNone        Packaging = "none"         // 无
	PackagingPlasticBox  Packaging = "plastic_box"  // 塑料盒
	PackagingPlasticBag  Packaging = "plastic_bag"  // 塑料袋
	PackagingCustomerOwn Packaging = "customer_own" // 自带容器
)

func (p Packaging) String() string {
	return string(p)
}

type PackagingMethod string

const (
	PackagingMethodNone          PackagingMethod = "none"            // 无
	PackagingMethodCombined      PackagingMethod = "combined"        // 装在一起
	PackagingMethodNoodleSoupSep PackagingMethod = "noodle_soup_sep" // 汤粉分开
)

func (p PackagingMethod) String() string {
	return string(p)
}
