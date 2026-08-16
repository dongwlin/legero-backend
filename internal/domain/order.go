package domain

import (
	"time"

	"github.com/google/uuid"
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

// Order is the domain model for a single order.
type Order struct {
	ID                   uuid.UUID
	WorkspaceID          uuid.UUID
	DisplayNo            string
	StapleTypeCode       *int16
	SizeCode             int16
	CustomSizePriceCents *int
	StapleAmountCode     int16
	ExtraStapleUnits     int16
	FriedEggCount        int16
	TofuSkewerCount      int16
	SelectedMeatCodes    []int16
	GreensCode           int16
	ScallionCode         int16
	PepperCode           int16
	DiningMethodCode     int16
	PackagingCode        *int16
	PackagingMethodCode  *int16
	TotalPriceCents      int
	StapleStepStatusCode int16
	MeatStepStatusCode   int16
	Note                 string
	CreatedBy            uuid.UUID
	UpdatedBy            uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
	CompletedAt          *time.Time
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
