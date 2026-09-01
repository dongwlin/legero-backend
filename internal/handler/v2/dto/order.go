package dto

import "github.com/dongwlin/legero-backend/internal/domain"

// ListOrdersResponseData is the paginated order list.
type ListOrdersResponseData struct {
	Items      []domain.OrderDTO `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

// CreateOrdersResponseData is the batch-create result.
type CreateOrdersResponseData struct {
	Items []domain.OrderDTO `json:"items"`
}

// UpdateOrderResponseData wraps a single updated order.
type UpdateOrderResponseData struct {
	Item domain.OrderDTO `json:"item"`
}

// ClearOrdersResponseData reports how many orders were cleared.
type ClearOrdersResponseData struct {
	ClearedCount int `json:"clearedCount"`
}
