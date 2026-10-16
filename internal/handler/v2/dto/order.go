package dto

import "time"

// OrderForm carries the v2 order form fields.
type OrderForm struct {
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

// CreateOrdersRequest carries a batch of identical orders.
type CreateOrdersRequest struct {
	Quantity int       `json:"quantity"`
	Form     OrderForm `json:"form"`
}

// UpdateOrderRequest carries replacement form data and concurrency tokens.
type UpdateOrderRequest struct {
	Form              OrderForm  `json:"form"`
	ExpectedVersion   *int64     `json:"expectedVersion"`
	ExpectedUpdatedAt *time.Time `json:"expectedUpdatedAt"`
}

// ToggleStepRequest selects a cooking step and carries concurrency tokens.
type ToggleStepRequest struct {
	Step              string     `json:"step"`
	ExpectedVersion   *int64     `json:"expectedVersion"`
	ExpectedUpdatedAt *time.Time `json:"expectedUpdatedAt"`
}

// ToggleServedRequest carries concurrency tokens for the served-state change.
type ToggleServedRequest struct {
	ExpectedVersion   *int64     `json:"expectedVersion"`
	ExpectedUpdatedAt *time.Time `json:"expectedUpdatedAt"`
}

// ClearOrdersRequest confirms which orders to clear.
type ClearOrdersRequest struct {
	Confirm bool   `json:"confirm"`
	Mode    string `json:"mode,omitempty"`
}

// Order is the v2 representation of an order.
type Order struct {
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

// ListOrdersResponseData is the paginated order list.
type ListOrdersResponseData struct {
	Items      []Order `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// CreateOrdersResponseData is the batch-create result.
type CreateOrdersResponseData struct {
	Items []Order `json:"items"`
}

// UpdateOrderResponseData wraps a single updated order.
type UpdateOrderResponseData struct {
	Item Order `json:"item"`
}

// ClearOrdersResponseData reports how many orders were cleared.
type ClearOrdersResponseData struct {
	ClearedCount int `json:"clearedCount"`
}
