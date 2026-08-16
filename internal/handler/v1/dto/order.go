package dto

import "github.com/dongwlin/legero-backend/internal/domain"

// ListOrdersResponse is the paginated order list.
type ListOrdersResponse struct {
	Items      []domain.OrderDTO `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

// CreateOrdersResponse is the batch-create result.
type CreateOrdersResponse struct {
	Items []domain.OrderDTO `json:"items"`
}

// UpdateOrderResponse wraps a single updated order.
type UpdateOrderResponse struct {
	Item domain.OrderDTO `json:"item"`
}

// ClearOrdersResponse reports how many orders were cleared.
type ClearOrdersResponse struct {
	ClearedCount int `json:"clearedCount"`
}
