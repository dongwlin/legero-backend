package domain

import (
	"time"

	"github.com/dongwlin/legero-backend/internal/infra/timex"
	"github.com/google/uuid"
)

const (
	EventOrderUpsert     = "order.upsert"
	EventOrderUpsertMany = "order.upsert_many"
	EventOrderDeleted    = "order.deleted"
	EventOrderCleared    = "order.cleared"
)

// Publisher is the interface for publishing order events.
type Publisher interface {
	Publish(workspaceID uuid.UUID, eventType string, payload any)
}

// UpsertEvent is the payload for an order upsert event.
type UpsertEvent struct {
	Item OrderDTO `json:"item"`
}

// UpsertManyEvent is the payload for a bounded batch of order upsert events.
// Keeping the item list bounded prevents a single realtime frame from growing
// without limit when a large create request is accepted.
type UpsertManyEvent struct {
	Items []OrderDTO `json:"items"`
}

// DeletedEvent is the payload for an order deleted event.
type DeletedEvent struct {
	ID string `json:"id"`
}

// ClearedEvent is the payload for a workspace cleared event.
type ClearedEvent struct {
	ClearedCount int                `json:"clearedCount"`
	Mode         ClearWorkspaceMode `json:"mode"`
	// ClearDateKey is the business-day cutoff (YYYY-MM-DD in the workspace
	// timezone) a before_today clear deleted orders before; empty for 'all'
	// clears. Clients pin their before_today barrier to this authoritative
	// server-computed key, so a delayed (cross-midnight) event or a skewed
	// client clock can never delete orders the server kept.
	ClearDateKey string `json:"clearDateKey,omitempty"`
}

// OrderDTO is the JSON-friendly representation of an Order for API responses and events.
type OrderDTO struct {
	ID                   string  `json:"id"`
	Version              int64   `json:"version"`
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
		Version:              o.Version,
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
