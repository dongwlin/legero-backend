package order

import "time"

type OrderDTO struct {
	ID                   string  `json:"id"`
	DisplayNo            string  `json:"displayNo"`
	StapleTypeCode       *int16  `json:"stapleTypeCode"`
	SizeCode             int16   `json:"sizeCode"`
	CustomSizePriceCents *int    `json:"customSizePriceCents"`
	StapleAmountCode     int16   `json:"stapleAmountCode"`
	ExtraStapleUnits     int16   `json:"extraStapleUnits"`
	SelectedMeatCodes    []int16 `json:"selectedMeatCodes"`
	GreensCode           int16   `json:"greensCode"`
	ScallionCode         int16   `json:"scallionCode"`
	PepperCode           int16   `json:"pepperCode"`
	DiningMethodCode     int16   `json:"diningMethodCode"`
	PackagingCode        *int16  `json:"packagingCode"`
	PackagingMethodCode  *int16  `json:"packagingMethodCode"`
	TotalPriceCents      int     `json:"totalPriceCents"`
	StapleStepStatusCode int16   `json:"stapleStepStatusCode"`
	MeatStepStatusCode   int16   `json:"meatStepStatusCode"`
	Note                 string  `json:"note"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
	CompletedAt          *string `json:"completedAt"`
}

func ToDTO(item Order, location *time.Location) OrderDTO {
	dto := OrderDTO{
		ID:                   item.ID.String(),
		DisplayNo:            item.DisplayNo,
		StapleTypeCode:       item.StapleTypeCode,
		SizeCode:             item.SizeCode,
		CustomSizePriceCents: item.CustomSizePriceCents,
		StapleAmountCode:     item.StapleAmountCode,
		ExtraStapleUnits:     item.ExtraStapleUnits,
		SelectedMeatCodes:    cloneInt16s(item.SelectedMeatCodes),
		GreensCode:           item.GreensCode,
		ScallionCode:         item.ScallionCode,
		PepperCode:           item.PepperCode,
		DiningMethodCode:     item.DiningMethodCode,
		PackagingCode:        item.PackagingCode,
		PackagingMethodCode:  item.PackagingMethodCode,
		TotalPriceCents:      item.TotalPriceCents,
		StapleStepStatusCode: item.StapleStepStatusCode,
		MeatStepStatusCode:   item.MeatStepStatusCode,
		Note:                 item.Note,
		CreatedAt:            formatTime(item.CreatedAt, location),
		UpdatedAt:            formatTime(item.UpdatedAt, location),
	}

	if item.CompletedAt != nil {
		value := formatTime(*item.CompletedAt, location)
		dto.CompletedAt = &value
	}

	return dto
}

func formatTime(value time.Time, location *time.Location) string {
	if location == nil {
		return value.Format(time.RFC3339)
	}
	return value.In(location).Format(time.RFC3339)
}

func cloneInt16s(values []int16) []int16 {
	if len(values) == 0 {
		return []int16{}
	}
	cloned := make([]int16, len(values))
	copy(cloned, values)
	return cloned
}
