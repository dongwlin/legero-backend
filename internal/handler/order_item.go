package handler

import (
	"time"

	"github.com/bytedance/sonic"
	"github.com/dongwlin/legero-backend/internal/handler/response"
	"github.com/dongwlin/legero-backend/internal/logic"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/pkg/broker"
	"github.com/dongwlin/legero-backend/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type OrderItem struct {
	broker         *broker.Broker
	orderItemLogic logic.OrderItem
}

func NewOrderItem(broker *broker.Broker, orderItemLogic logic.OrderItem) *OrderItem {
	return &OrderItem{
		broker:         broker,
		orderItemLogic: orderItemLogic,
	}
}

func (h *OrderItem) RegisterRoutes(r fiber.Router) {

	orderItem := r.Group("/orderItems")

	orderItem.Post("/", h.Create)
	orderItem.Get("/", h.List)
}

type OrderItemCreateRequest struct {
	IncludeNoodles    bool                  `json:"include_noodles" validate:"required"`
	NoodleType        types.Noodle          `json:"noodle_type" validate:"required"`
	CustomNoodleType  string                `json:"custom_noodle_type" validate:"required_if=NoodleType custom"`
	NoodleAmount      types.Adjustment      `json:"noodle_amount" validate:"required"`
	ExtraNoodleBlocks int                   `json:"extra_noodle_blocks" validate:"required"`
	Size              types.Size            `json:"size" validate:"required"`
	CustomSizePrice   float64               `json:"custom_size_price" validate:"required_if=Size custom"`
	MeatAvailable     []types.Meat          `json:"meat_available" validate:"required"`
	MeatExcluded      []types.Meat          `json:"meat_excluded" validate:"required"`
	Greens            types.Adjustment      `json:"greens" validate:"required"`
	Scallion          types.Adjustment      `json:"scallion" validate:"required"`
	Pepper            types.Adjustment      `json:"pepper" validate:"required"`
	DiningMethod      types.DiningMethod    `json:"dining_method" validate:"required"`
	Packaging         types.Packaging       `json:"packaging" validate:"required"`
	PackagingMethod   types.PackagingMethod `json:"packaging_method" validate:"required"`
	Note              string                `json:"note"`
}

type OrderItemCreateResponseData struct {
	DailyID           uint64                `json:"daily_id"`
	IncludeNoodles    bool                  `json:"include_noodles"`
	NoodleType        types.Noodle          `json:"noodle_type"`
	CustomNoodleType  string                `json:"custom_noodle_type"`
	NoodleAmount      types.Adjustment      `json:"noodle_amount"`
	ExtraNoodleBlocks int                   `json:"extra_noodle_blocks"`
	Size              types.Size            `json:"size"`
	CustomSizePrice   float64               `json:"custom_size_price"`
	MeatAvailable     []types.Meat          `json:"meat_available"`
	MeatExcluded      []types.Meat          `json:"meat_excluded"`
	Greens            types.Adjustment      `json:"greens"`
	Scallion          types.Adjustment      `json:"scallion"`
	Pepper            types.Adjustment      `json:"pepper"`
	DiningMethod      types.DiningMethod    `json:"dining_method"`
	Packaging         types.Packaging       `json:"packaging"`
	PackagingMethod   types.PackagingMethod `json:"packaging_method"`
	Note              string                `json:"note"`
	Price             float64               `json:"price"`
	ProgressNoodles   types.StepStatus      `json:"progress_noodles"`
	ProgressMeat      types.StepStatus      `json:"progress_meat"`
	CompletedAt       time.Time             `json:"completed_at"`
	CreatedAt         time.Time             `json:"created_at"`
}

func (h *OrderItem) Create(c *fiber.Ctx) error {

	var req OrderItemCreateRequest
	if err := validator.ValidateBody(c, &req); err != nil {
		return err
	}

	result, err := h.orderItemLogic.Create(c.UserContext(), logic.OrderItemCreateParams{
		IncludeNoodles:    req.IncludeNoodles,
		NoodleType:        req.NoodleType,
		CustomNoodleType:  req.CustomNoodleType,
		NoodleAmount:      req.NoodleAmount,
		ExtraNoodleBlocks: req.ExtraNoodleBlocks,
		Size:              req.Size,
		CustomSizePrice:   req.CustomSizePrice,
		MeatAvailable:     req.MeatAvailable,
		MeatExcluded:      req.MeatExcluded,
		Greens:            req.Greens,
		Scallion:          req.Scallion,
		Pepper:            req.Pepper,
		DiningMethod:      req.DiningMethod,
		Packaging:         req.Packaging,
		PackagingMethod:   req.PackagingMethod,
		Note:              req.Note,
	})

	if err != nil {

		log.Error().
			Err(err).
			Msg("failed to create order item")

		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	data := OrderItemCreateResponseData{
		DailyID:           result.DailyID,
		IncludeNoodles:    result.IncludeNoodles,
		NoodleType:        result.NoodleType,
		CustomNoodleType:  result.CustomNoodleType,
		NoodleAmount:      result.NoodleAmount,
		ExtraNoodleBlocks: result.ExtraNoodleBlocks,
		Size:              result.Size,
		CustomSizePrice:   result.CustomSizePrice,
		MeatAvailable:     result.MeatAvailable,
		MeatExcluded:      result.MeatExcluded,
		Greens:            result.Greens,
		Scallion:          result.Scallion,
		Pepper:            result.Pepper,
		DiningMethod:      result.DiningMethod,
		Packaging:         result.Packaging,
		PackagingMethod:   result.PackagingMethod,
		Note:              result.Note,
		Price:             result.Price,
		ProgressNoodles:   result.ProgressNoodles,
		ProgressMeat:      result.ProgressMeat,
		CompletedAt:       result.CompletedAt,
		CreatedAt:         result.CreatedAt,
	}

	msgData, err := sonic.Marshal(data)
	if err != nil {

		log.Error().
			Err(err).
			Msg("failed to marshal order item data")

		resp := response.UnexpectedError()
		return c.Status(fiber.StatusInternalServerError).JSON(resp)
	}

	h.broker.Broadcast(msgData)

	resp := response.Success(data)
	return c.JSON(resp)
}

func (h *OrderItem) List(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotImplemented)
}
