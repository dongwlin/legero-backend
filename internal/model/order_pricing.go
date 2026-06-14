package model

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

// GetBasePriceCents returns the base price in cents for the given order form input.
func (f OrderFormInput) GetBasePriceCents() int {
	if f.SizeCode == SizeCustom {
		if f.CustomSizePriceCents == nil {
			return 0
		}
		if *f.CustomSizePriceCents < 0 {
			return 0
		}
		return *f.CustomSizePriceCents
	}

	if f.StapleTypeCode != nil && *f.StapleTypeCode == StapleTypeRice {
		if f.SizeCode == SizeLarge {
			return riceLargeBasePriceCents
		}
		return riceMediumBasePriceCents
	}

	if f.StapleTypeCode != nil && *f.StapleTypeCode == StapleTypeYiNoodle {
		switch f.SizeCode {
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

	switch f.SizeCode {
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

// CalculateTotalPriceCents computes the total price in cents for the given order form input,
// including extras (extra staple units, fried egg, tofu skewer, packaging).
func (f OrderFormInput) CalculateTotalPriceCents() int {
	total := f.GetBasePriceCents()

	if f.StapleTypeCode != nil && *f.StapleTypeCode == StapleTypeYiNoodle {
		total += int(f.ExtraStapleUnits) * ExtraStapleUnitPriceCents
	}

	total += int(f.FriedEggCount) * FriedEggPriceCents
	total += int(f.TofuSkewerCount) * TofuSkewerPriceCents

	if f.DiningMethodCode == DiningMethodTakeout && f.PackagingCode != nil && *f.PackagingCode == PackagingContainer {
		total += PlasticContainerPriceCents
	}

	return total
}
