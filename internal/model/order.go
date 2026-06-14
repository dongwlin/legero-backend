package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Staple type codes.
const (
	StapleTypeRiceSheet      int16 = 1
	StapleTypeRiceVermicelli int16 = 2
	StapleTypeYiNoodle       int16 = 3
	StapleTypeRice           int16 = 4
)

// Size codes.
const (
	SizeSmall  int16 = 1
	SizeMedium int16 = 2
	SizeLarge  int16 = 3
	SizeCustom int16 = 4
)

// Adjustment codes (used for staple amount, greens, scallion, pepper).
const (
	AdjustmentNormal int16 = 1
	AdjustmentLess   int16 = 2
	AdjustmentMore   int16 = 3
	AdjustmentNone   int16 = 4
)

// Dining method codes.
const (
	DiningMethodDineIn  int16 = 1
	DiningMethodTakeout int16 = 2
)

// Packaging codes.
const (
	PackagingContainer int16 = 1
	PackagingBag       int16 = 2
)

// Packaging method codes.
const (
	PackagingMethodTogether  int16 = 1
	PackagingMethodSeparated int16 = 2
)

// Step status codes.
const (
	StepStatusUnrequired int16 = 1
	StepStatusNotStarted int16 = 2
	StepStatusCompleted  int16 = 3
)

// Meat codes.
const (
	MeatLeanPork       int16 = 1
	MeatLiver          int16 = 2
	MeatBloodCurd      int16 = 3
	MeatLargeIntestine int16 = 4
	MeatSmallIntestine int16 = 5
	MeatKidney         int16 = 6
)

var (
	allStapleTypeCodes      = []int16{StapleTypeRiceSheet, StapleTypeRiceVermicelli, StapleTypeYiNoodle, StapleTypeRice}
	allSizeCodes            = []int16{SizeSmall, SizeMedium, SizeLarge, SizeCustom}
	allAdjustmentCodes      = []int16{AdjustmentNormal, AdjustmentLess, AdjustmentMore, AdjustmentNone}
	allDiningMethodCodes    = []int16{DiningMethodDineIn, DiningMethodTakeout}
	allPackagingCodes       = []int16{PackagingContainer, PackagingBag}
	allPackagingMethodCodes = []int16{PackagingMethodTogether, PackagingMethodSeparated}
	allMeatCodes            = []int16{MeatLeanPork, MeatLiver, MeatBloodCurd, MeatLargeIntestine, MeatSmallIntestine, MeatKidney}
)

// ListStatus filters orders by completion status.
type ListStatus string

const (
	ListStatusUncompleted ListStatus = "uncompleted"
	ListStatusCompleted   ListStatus = "completed"
	ListStatusAll         ListStatus = "all"
)

// Valid reports whether the ListStatus value is one of the defined constants.
func (s ListStatus) Valid() bool {
	return s == ListStatusUncompleted || s == ListStatusCompleted || s == ListStatusAll
}

// Actor identifies the user performing an order action.
type Actor struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Role        Role
}

// OrderFormInput carries the user-submitted form data for creating or updating an order.
type OrderFormInput struct {
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
	Note                 string  `json:"note"`
}

// Order is the unified domain + ORM model for the orders table.
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

// ListOrdersQuery carries parameters for listing orders.
type ListOrdersQuery struct {
	Status ListStatus
	Limit  int
	Cursor string
}

// ListOrdersResult is the paginated result of a list-orders query.
type ListOrdersResult struct {
	Items      []Order
	NextCursor *string
}

// ToggleStepInput carries the payload for toggling a cooking step.
type ToggleStepInput struct {
	Step              string     `json:"step"`
	ExpectedUpdatedAt *time.Time `json:"expectedUpdatedAt"`
}

// ToggleServedInput carries the payload for toggling the served state.
type ToggleServedInput struct {
	ExpectedUpdatedAt *time.Time `json:"expectedUpdatedAt"`
}

// UpdateOrderInput carries the payload for updating an existing order.
type UpdateOrderInput struct {
	Form              OrderFormInput `json:"form"`
	ExpectedUpdatedAt *time.Time     `json:"expectedUpdatedAt"`
}

// CreateOrdersInput carries the payload for batch-creating orders.
type CreateOrdersInput struct {
	Quantity int            `json:"quantity"`
	Form     OrderFormInput `json:"form"`
}

// ClearWorkspaceMode controls which orders are cleared.
type ClearWorkspaceMode string

const (
	ClearWorkspaceModeAll         ClearWorkspaceMode = "all"
	ClearWorkspaceModeBeforeToday ClearWorkspaceMode = "before_today"
)

// ClearWorkspaceInput carries the payload for clearing a workspace's orders.
type ClearWorkspaceInput struct {
	Confirm bool               `json:"confirm"`
	Mode    ClearWorkspaceMode `json:"mode,omitempty"`
}

// Normalize returns ClearWorkspaceModeAll when the mode is empty.
func (m ClearWorkspaceMode) Normalize() ClearWorkspaceMode {
	if m == "" {
		return ClearWorkspaceModeAll
	}
	return m
}

// Valid reports whether the mode is a recognized value.
func (m ClearWorkspaceMode) Valid() bool {
	switch m.Normalize() {
	case ClearWorkspaceModeAll, ClearWorkspaceModeBeforeToday:
		return true
	default:
		return false
	}
}
