package order

import (
	"strings"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
)

func NormalizeForm(input OrderFormInput) (OrderFormInput, error) {
	if input.ExtraStapleUnits < 0 {
		return OrderFormInput{}, httpx.ValidationError("extraStapleUnits must be greater than or equal to 0")
	}
	if input.FriedEggCount < 0 {
		return OrderFormInput{}, httpx.ValidationError("friedEggCount must be greater than or equal to 0")
	}
	if input.TofuSkewerCount < 0 {
		return OrderFormInput{}, httpx.ValidationError("tofuSkewerCount must be greater than or equal to 0")
	}

	if input.StapleTypeCode != nil && !containsInt16(allStapleTypeCodes, *input.StapleTypeCode) {
		return OrderFormInput{}, httpx.ValidationError("stapleTypeCode is invalid")
	}
	if !containsInt16(allSizeCodes, input.SizeCode) {
		return OrderFormInput{}, httpx.ValidationError("sizeCode is invalid")
	}
	if !containsInt16(allAdjustmentCodes, input.StapleAmountCode) {
		return OrderFormInput{}, httpx.ValidationError("stapleAmountCode is invalid")
	}
	if !containsInt16(allAdjustmentCodes, input.GreensCode) {
		return OrderFormInput{}, httpx.ValidationError("greensCode is invalid")
	}
	if !containsInt16(allAdjustmentCodes, input.ScallionCode) {
		return OrderFormInput{}, httpx.ValidationError("scallionCode is invalid")
	}
	if !containsInt16(allAdjustmentCodes, input.PepperCode) {
		return OrderFormInput{}, httpx.ValidationError("pepperCode is invalid")
	}
	if !containsInt16(allDiningMethodCodes, input.DiningMethodCode) {
		return OrderFormInput{}, httpx.ValidationError("diningMethodCode is invalid")
	}
	if input.PackagingCode != nil && !containsInt16(allPackagingCodes, *input.PackagingCode) {
		return OrderFormInput{}, httpx.ValidationError("packagingCode is invalid")
	}
	if input.PackagingMethodCode != nil && !containsInt16(allPackagingMethodCodes, *input.PackagingMethodCode) {
		return OrderFormInput{}, httpx.ValidationError("packagingMethodCode is invalid")
	}
	for _, code := range input.SelectedMeatCodes {
		if !containsInt16(allMeatCodes, code) {
			return OrderFormInput{}, httpx.ValidationError("selectedMeatCodes contains invalid value")
		}
	}

	normalized := input
	normalized.Note = strings.TrimSpace(input.Note)

	if normalized.StapleTypeCode != nil && *normalized.StapleTypeCode == StapleTypeRice && normalized.SizeCode == SizeSmall {
		normalized.SizeCode = SizeMedium
	}

	if normalized.SizeCode == SizeCustom {
		if normalized.CustomSizePriceCents == nil || *normalized.CustomSizePriceCents <= 0 {
			return OrderFormInput{}, httpx.ValidationError("customSizePriceCents must be set when sizeCode is custom")
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
		return OrderFormInput{}, httpx.ValidationError("at least one of stapleTypeCode or selectedMeatCodes is required")
	}

	return normalized, nil
}

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

func containsInt16(values []int16, target int16) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
