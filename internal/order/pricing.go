package order

const (
	ExtraStapleUnitPriceCents    = 300
	FriedEggPriceCents           = 200
	TofuSkewerPriceCents         = 200
	PlasticContainerPriceCents   = 50
	defaultSmallBasePriceCents   = 1000
	defaultMediumBasePriceCents  = 1200
	defaultLargeBasePriceCents   = 1500
	yiNoodleSmallBasePriceCents  = 1100
	yiNoodleMediumBasePriceCents = 1300
	yiNoodleLargeBasePriceCents  = 1600
	riceMediumBasePriceCents     = 1500
	riceLargeBasePriceCents      = 2000
)

func GetBasePriceCents(input OrderFormInput) int {
	if input.SizeCode == SizeCustom {
		if input.CustomSizePriceCents == nil {
			return 0
		}
		if *input.CustomSizePriceCents < 0 {
			return 0
		}
		return *input.CustomSizePriceCents
	}

	if input.StapleTypeCode != nil && *input.StapleTypeCode == StapleTypeRice {
		if input.SizeCode == SizeLarge {
			return riceLargeBasePriceCents
		}
		return riceMediumBasePriceCents
	}

	if input.StapleTypeCode != nil && *input.StapleTypeCode == StapleTypeYiNoodle {
		switch input.SizeCode {
		case SizeSmall:
			return yiNoodleSmallBasePriceCents
		case SizeMedium:
			return yiNoodleMediumBasePriceCents
		case SizeLarge:
			return yiNoodleLargeBasePriceCents
		default:
			return 0
		}
	}

	switch input.SizeCode {
	case SizeSmall:
		return defaultSmallBasePriceCents
	case SizeMedium:
		return defaultMediumBasePriceCents
	case SizeLarge:
		return defaultLargeBasePriceCents
	default:
		return 0
	}
}

func CalculateTotalPriceCents(input OrderFormInput) int {
	total := GetBasePriceCents(input)

	if input.StapleTypeCode != nil && *input.StapleTypeCode == StapleTypeYiNoodle {
		total += int(input.ExtraStapleUnits) * ExtraStapleUnitPriceCents
	}

	total += int(input.FriedEggCount) * FriedEggPriceCents
	total += int(input.TofuSkewerCount) * TofuSkewerPriceCents

	if input.DiningMethodCode == DiningMethodTakeout && input.PackagingCode != nil && *input.PackagingCode == PackagingContainer {
		total += PlasticContainerPriceCents
	}

	return total
}
