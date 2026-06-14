package model

import (
	"time"

	"github.com/dongwlin/legero-backend/internal/infra/timex"
	"github.com/google/uuid"
)

const (
	EventOrderUpsert  = "order.upsert"
	EventOrderDeleted = "order.deleted"
	EventOrderCleared = "order.cleared"
)

// Publisher is the interface for publishing order events.
type Publisher interface {
	Publish(workspaceID uuid.UUID, eventType string, payload any)
}

// UpsertEvent is the payload for an order upsert event.
type UpsertEvent struct {
	Item OrderDTO `json:"item"`
}

// DeletedEvent is the payload for an order deleted event.
type DeletedEvent struct {
	ID string `json:"id"`
}

// ClearedEvent is the payload for a workspace cleared event.
type ClearedEvent struct {
	ClearedCount int                `json:"clearedCount"`
	Mode         ClearWorkspaceMode `json:"mode"`
}

// OrderDTO is the JSON-friendly representation of an Order for API responses and events.
type OrderDTO struct {
	ID                   string  `json:"id"`
	DisplayNo            string  `json:"displayNo"`
	StapleTypeCode       *int16  `json:"stapleTypeCode"`
	SizeCode             int16   `json:"sizeCode"`
	CustomSizePriceCents *int    `json:"customSizePriceCents"`
	StapleAmountCode     int16   `json:"stapleAmountCode"`
	ExtraStapleUnits     int16   `json:"extraStapleUnits"`
	FriedEggCount        int16   `json:"friedEggCount"`
	TofuSkewerCount      int16   `json:"tofuSkewerCount"`
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

// ToDTO converts an Order to an OrderDTO, formatting times in the given location.
func (o Order) ToDTO(location *time.Location) OrderDTO {
	dto := OrderDTO{
		ID:                   o.ID.String(),
		DisplayNo:            o.DisplayNo,
		StapleTypeCode:       o.StapleTypeCode,
		SizeCode:             o.SizeCode,
		CustomSizePriceCents: o.CustomSizePriceCents,
		StapleAmountCode:     o.StapleAmountCode,
		ExtraStapleUnits:     o.ExtraStapleUnits,
		FriedEggCount:        o.FriedEggCount,
		TofuSkewerCount:      o.TofuSkewerCount,
		SelectedMeatCodes:    CloneInt16s(o.SelectedMeatCodes),
		GreensCode:           o.GreensCode,
		ScallionCode:         o.ScallionCode,
		PepperCode:           o.PepperCode,
		DiningMethodCode:     o.DiningMethodCode,
		PackagingCode:        o.PackagingCode,
		PackagingMethodCode:  o.PackagingMethodCode,
		TotalPriceCents:      o.TotalPriceCents,
		StapleStepStatusCode: o.StapleStepStatusCode,
		MeatStepStatusCode:   o.MeatStepStatusCode,
		Note:                 o.Note,
		CreatedAt:            timex.FormatTime(o.CreatedAt, location),
		UpdatedAt:            timex.FormatTime(o.UpdatedAt, location),
	}

	if o.CompletedAt != nil {
		value := timex.FormatTime(*o.CompletedAt, location)
		dto.CompletedAt = &value
	}

	return dto
}

