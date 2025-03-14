package model

import (
	"time"

	"github.com/dongwlin/legero-backend/internal/ent"
	"github.com/dongwlin/legero-backend/internal/model/types"
)

type OrderItem struct {
	ID      uint64 `json:"id"`
	DailyID uint64 `json:"daily_id"`

	IncludeNoodles    bool             `json:"include_noodles"`
	NoodleType        types.Noodle     `json:"noodle_type"`
	CustomNoodleType  string           `json:"custom_noodle_type"`
	NoodleAmount      types.Adjustment `json:"noodle_amount"`
	ExtraNoodleBlocks int              `json:"extra_noodle_blocks"`

	Size            types.Size `json:"size"`
	CustomSizePrice float64    `json:"custom_size_price"`

	MeatAvailable []types.Meat `json:"meat_available"`
	MeatExcluded  []types.Meat `json:"meat_excluded"`

	Greens   types.Adjustment `json:"greens"`
	Scallion types.Adjustment `json:"scallion"`
	Pepper   types.Adjustment `json:"pepper"`

	DiningMethod    types.DiningMethod    `json:"dining_method"`
	Packaging       types.Packaging       `json:"packaging"`
	PackagingMethod types.PackagingMethod `json:"packaging_method"`

	Note  string  `json:"note"`
	Price float64 `json:"price"`

	ProgressNoodles types.StepStatus `json:"progress_noodles"`
	ProgressMeat    types.StepStatus `json:"progress_meat"`

	CompletedAt time.Time `json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func EntToOrderItem(e *ent.OrderItem) *OrderItem {
	return &OrderItem{
		ID:                e.ID,
		DailyID:           e.DailyID,
		IncludeNoodles:    e.IncludeNoodles,
		NoodleType:        types.Noodle(e.NoodleType),
		CustomNoodleType:  e.CustomNoodleType,
		NoodleAmount:      types.Adjustment(e.NoodleAmount),
		ExtraNoodleBlocks: e.ExtraNoodleBlocks,
		Size:              types.Size(e.Size),
		CustomSizePrice:   e.CustomSizePrice,
		MeatAvailable:     types.StringsToMeats(e.MeatAvailable),
		MeatExcluded:      types.StringsToMeats(e.MeatExcluded),
		Greens:            types.Adjustment(e.Greens),
		Scallion:          types.Adjustment(e.Scallion),
		Pepper:            types.Adjustment(e.Pepper),
		DiningMethod:      types.DiningMethod(e.DiningMethod),
		Packaging:         types.Packaging(e.Packaging),
		PackagingMethod:   types.PackagingMethod(e.PackagingMethod),
		Note:              e.Note,
		Price:             e.Price,
		ProgressNoodles:   types.StepStatus(e.ProgressNoodles),
		ProgressMeat:      types.StepStatus(e.ProgressMeat),
		CompletedAt:       e.CompletedAt,
		CreatedAt:         e.CreatedAt,
	}
}
