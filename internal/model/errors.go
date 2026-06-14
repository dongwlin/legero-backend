package model

import "errors"

// Domain errors for order validation.
var (
	ErrNegativeExtraStapleUnits = errors.New("extraStapleUnits must be greater than or equal to 0")
	ErrNegativeFriedEggCount    = errors.New("friedEggCount must be greater than or equal to 0")
	ErrNegativeTofuSkewerCount  = errors.New("tofuSkewerCount must be greater than or equal to 0")
	ErrInvalidStapleTypeCode    = errors.New("stapleTypeCode is invalid")
	ErrInvalidSizeCode          = errors.New("sizeCode is invalid")
	ErrInvalidStapleAmountCode  = errors.New("stapleAmountCode is invalid")
	ErrInvalidGreensCode        = errors.New("greensCode is invalid")
	ErrInvalidScallionCode      = errors.New("scallionCode is invalid")
	ErrInvalidPepperCode        = errors.New("pepperCode is invalid")
	ErrInvalidDiningMethodCode  = errors.New("diningMethodCode is invalid")
	ErrInvalidPackagingCode     = errors.New("packagingCode is invalid")
	ErrInvalidPackagingMethod   = errors.New("packagingMethodCode is invalid")
	ErrInvalidMeatCode          = errors.New("selectedMeatCodes contains invalid value")
	ErrMissingCustomSizePrice   = errors.New("customSizePriceCents must be set when sizeCode is custom")
	ErrEmptyOrder               = errors.New("at least one of stapleTypeCode or selectedMeatCodes is required")
)
