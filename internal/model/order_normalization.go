package model

import (
	"strings"
)

// Normalize validates and normalizes an OrderFormInput.
// Returns the normalized input or a descriptive error.
func (f OrderFormInput) Normalize() (OrderFormInput, error) {
	if f.ExtraStapleUnits < 0 {
		return OrderFormInput{}, ErrNegativeExtraStapleUnits
	}
	if f.FriedEggCount < 0 {
		return OrderFormInput{}, ErrNegativeFriedEggCount
	}
	if f.TofuSkewerCount < 0 {
		return OrderFormInput{}, ErrNegativeTofuSkewerCount
	}

	if f.StapleTypeCode != nil && !containsInt16(allStapleTypeCodes, *f.StapleTypeCode) {
		return OrderFormInput{}, ErrInvalidStapleTypeCode
	}
	if !containsInt16(allSizeCodes, f.SizeCode) {
		return OrderFormInput{}, ErrInvalidSizeCode
	}
	if !containsInt16(allAdjustmentCodes, f.StapleAmountCode) {
		return OrderFormInput{}, ErrInvalidStapleAmountCode
	}
	if !containsInt16(allAdjustmentCodes, f.GreensCode) {
		return OrderFormInput{}, ErrInvalidGreensCode
	}
	if !containsInt16(allAdjustmentCodes, f.ScallionCode) {
		return OrderFormInput{}, ErrInvalidScallionCode
	}
	if !containsInt16(allAdjustmentCodes, f.PepperCode) {
		return OrderFormInput{}, ErrInvalidPepperCode
	}
	if !containsInt16(allDiningMethodCodes, f.DiningMethodCode) {
		return OrderFormInput{}, ErrInvalidDiningMethodCode
	}
	if f.PackagingCode != nil && !containsInt16(allPackagingCodes, *f.PackagingCode) {
		return OrderFormInput{}, ErrInvalidPackagingCode
	}
	if f.PackagingMethodCode != nil && !containsInt16(allPackagingMethodCodes, *f.PackagingMethodCode) {
		return OrderFormInput{}, ErrInvalidPackagingMethod
	}
	for _, code := range f.SelectedMeatCodes {
		if !containsInt16(allMeatCodes, code) {
			return OrderFormInput{}, ErrInvalidMeatCode
		}
	}

	normalized := f
	normalized.Note = strings.TrimSpace(f.Note)

	if normalized.StapleTypeCode != nil && *normalized.StapleTypeCode == StapleTypeRice && normalized.SizeCode == SizeSmall {
		normalized.SizeCode = SizeMedium
	}

	if normalized.SizeCode == SizeCustom {
		if normalized.CustomSizePriceCents == nil || *normalized.CustomSizePriceCents <= 0 {
			return OrderFormInput{}, ErrMissingCustomSizePrice
		}
	} else {
		normalized.CustomSizePriceCents = nil
	}

	if normalized.StapleTypeCode == nil {
		normalized.StapleAmountCode = AdjustmentNormal
	}

	if normalized.StapleTypeCode == nil || *normalized.StapleTypeCode != StapleTypeYiNoodle {
		normalized.ExtraStapleUnits = 0
	}

	normalized.SelectedMeatCodes = normalizeSelectedMeatCodes(normalized.SelectedMeatCodes, normalized.SizeCode)

	if normalized.DiningMethodCode == DiningMethodDineIn {
		normalized.PackagingCode = nil
		normalized.PackagingMethodCode = nil
	} else {
		if normalized.PackagingCode == nil {
			value := int16(PackagingContainer)
			normalized.PackagingCode = &value
		}
		if normalized.PackagingMethodCode == nil {
			defaultMethod := int16(PackagingMethodTogether)
			if normalized.StapleTypeCode != nil && *normalized.StapleTypeCode == StapleTypeRice {
				defaultMethod = PackagingMethodSeparated
			}
			normalized.PackagingMethodCode = &defaultMethod
		}
	}

	if normalized.StapleTypeCode == nil && len(normalized.SelectedMeatCodes) == 0 {
		return OrderFormInput{}, ErrEmptyOrder
	}

	return normalized, nil
}

// normalizeSelectedMeatCodes filters meat codes to only those visible for the given size,
// preserving the original selection order.
func normalizeSelectedMeatCodes(codes []int16, sizeCode int16) []int16 {
	allowedCodes := visibleMeatCodes(sizeCode)
	selected := make(map[int16]struct{}, len(codes))
	for _, code := range codes {
		selected[code] = struct{}{}
	}

	normalized := make([]int16, 0, len(allowedCodes))
	for _, code := range allowedCodes {
		if _, ok := selected[code]; ok {
			normalized = append(normalized, code)
		}
	}
	return normalized
}

// visibleMeatCodes returns the meat codes available for the given size.
// Small size excludes kidney; all other sizes include all meats.
func visibleMeatCodes(sizeCode int16) []int16 {
	if sizeCode == SizeSmall {
		return []int16{
			MeatLeanPork,
			MeatLiver,
			MeatBloodCurd,
			MeatLargeIntestine,
			MeatSmallIntestine,
		}
	}

	return []int16{
		MeatLeanPork,
		MeatLiver,
		MeatBloodCurd,
		MeatLargeIntestine,
		MeatSmallIntestine,
		MeatKidney,
	}
}

// containsInt16 reports whether target is present in values.
func containsInt16(values []int16, target int16) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
