package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Order is the bun ORM mapping of the orders table.
type Order struct {
	bun.BaseModel `bun:"table:orders,alias:o"`

	ID                   uuid.UUID  `bun:",pk,type:uuid"`
	WorkspaceID          uuid.UUID  `bun:"workspace_id,type:uuid,notnull"`
	DisplayNo            string     `bun:"display_no,notnull"`
	StapleTypeCode       *int16     `bun:"staple_type_code"`
	SizeCode             int16      `bun:"size_code,notnull"`
	CustomSizePriceCents *int       `bun:"custom_size_price_cents"`
	StapleAmountCode     int16      `bun:"staple_amount_code,notnull"`
	ExtraStapleUnits     int16      `bun:"extra_staple_units,notnull"`
	FriedEggCount        int16      `bun:"fried_egg_count,notnull"`
	TofuSkewerCount      int16      `bun:"tofu_skewer_count,notnull"`
	SelectedMeatCodes    []int16    `bun:"selected_meat_codes,array,type:smallint[],notnull"`
	GreensCode           int16      `bun:"greens_code,notnull"`
	ScallionCode         int16      `bun:"scallion_code,notnull"`
	PepperCode           int16      `bun:"pepper_code,notnull"`
	DiningMethodCode     int16      `bun:"dining_method_code,notnull"`
	PackagingCode        *int16     `bun:"packaging_code"`
	PackagingMethodCode  *int16     `bun:"packaging_method_code"`
	TotalPriceCents      int        `bun:"total_price_cents,notnull"`
	StapleStepStatusCode int16      `bun:"staple_step_status_code,notnull"`
	MeatStepStatusCode   int16      `bun:"meat_step_status_code,notnull"`
	Note                 string     `bun:"note,notnull"`
	CreatedBy            uuid.UUID  `bun:"created_by,type:uuid,notnull"`
	UpdatedBy            uuid.UUID  `bun:"updated_by,type:uuid,notnull"`
	CreatedAt            time.Time  `bun:"created_at,notnull"`
	UpdatedAt            time.Time  `bun:"updated_at,notnull"`
	CompletedAt          *time.Time `bun:"completed_at"`
}
