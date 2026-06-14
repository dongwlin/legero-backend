package handler

import (
	"time"

	"github.com/dongwlin/legero-backend/internal/model"
)

// toOrderDTOs converts a slice of Order to a slice of OrderDTO.
func toOrderDTOs(items []model.Order, location *time.Location) []model.OrderDTO {
	dtos := make([]model.OrderDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, item.ToDTO(location))
	}
	return dtos
}
