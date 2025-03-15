package logic

import (
	"context"
	"time"

	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/model/types"
	"github.com/dongwlin/legero-backend/internal/pkg/dailyid"
	"github.com/dongwlin/legero-backend/internal/pkg/errs"
	"github.com/dongwlin/legero-backend/internal/repo"
)

type OrderItem interface {
	Create(ctx context.Context, params OrderItemCreateParams) (*OrderItemInfo, error)
	List(ctx context.Context, page, size int) ([]*OrderItemInfo, int64, error)
	UpdateRequires(ctx context.Context, params OrderItemUpdateRequiresParams) (*OrderItemInfo, error)
	UpdateProgress(ctx context.Context, params OrderItemUpdateProgressParams) (*OrderItemInfo, error)
}

type (
	OrderItemCreateParams struct {
		IncludeNoodles    bool
		NoodleType        types.Noodle
		CustomNoodleType  string
		NoodleAmount      types.Adjustment
		ExtraNoodleBlocks int
		Size              types.Size
		CustomSizePrice   float64
		MeatAvailable     []types.Meat
		MeatExcluded      []types.Meat
		Greens            types.Adjustment
		Scallion          types.Adjustment
		Pepper            types.Adjustment
		DiningMethod      types.DiningMethod
		Packaging         types.Packaging
		PackagingMethod   types.PackagingMethod
		Note              string
	}

	OrderItemUpdateRequiresParams struct {
		DisplayID         string
		IncludeNoodles    bool
		NoodleType        types.Noodle
		CustomNoodleType  string
		NoodleAmount      types.Adjustment
		ExtraNoodleBlocks int
		Size              types.Size
		CustomSizePrice   float64
		MeatAvailable     []types.Meat
		MeatExcluded      []types.Meat
		Greens            types.Adjustment
		Scallion          types.Adjustment
		Pepper            types.Adjustment
		DiningMethod      types.DiningMethod
		Packaging         types.Packaging
		PackagingMethod   types.PackagingMethod
		Note              string
	}

	OrderItemUpdateProgressParams struct {
		DisplayID       string
		ProgressNoodles types.StepStatus
		ProgressMeat    types.StepStatus
	}
)

type (
	OrderItemInfo struct {
		DailyID           uint64
		IncludeNoodles    bool
		NoodleType        types.Noodle
		CustomNoodleType  string
		NoodleAmount      types.Adjustment
		ExtraNoodleBlocks int
		Size              types.Size
		CustomSizePrice   float64
		MeatAvailable     []types.Meat
		MeatExcluded      []types.Meat
		Greens            types.Adjustment
		Scallion          types.Adjustment
		Pepper            types.Adjustment
		DiningMethod      types.DiningMethod
		Packaging         types.Packaging
		PackagingMethod   types.PackagingMethod
		Note              string
		Price             float64
		ProgressNoodles   types.StepStatus
		ProgressMeat      types.StepStatus
		CompletedAt       time.Time
		CreatedAt         time.Time
	}
)

var PriceMap = map[string]map[string]float64{
	types.NoodleNone.String():     {types.SizeNone.String(): 0, types.SizeSmall.String(): 0, types.SizeMiddle.String(): 0, types.SizeLarge.String(): 0},
	types.NoodleFlatRice.String(): {types.SizeNone.String(): 0, types.SizeSmall.String(): 10, types.SizeMiddle.String(): 12, types.SizeLarge.String(): 15},
	types.NoodleThinRice.String(): {types.SizeNone.String(): 0, types.SizeSmall.String(): 10, types.SizeMiddle.String(): 12, types.SizeLarge.String(): 15},
	types.NoodleYi.String():       {types.SizeNone.String(): 0, types.SizeSmall.String(): 11, types.SizeMiddle.String(): 13, types.SizeLarge.String(): 16},
	types.NoodleCustom.String():   {types.SizeNone.String(): 0, types.SizeSmall.String(): 10, types.SizeMiddle.String(): 12, types.SizeLarge.String(): 15},
}

const (
	YiNoodlePrice   = 3
	PlasticBoxPrice = 0.5
)

type OrderItemImpl struct {
	dailyIDGenerator *dailyid.DailyIDGenerator
	orderItemRepo    repo.OrderItem
}

// Create implements OrderItem.
func (l *OrderItemImpl) Create(ctx context.Context, params OrderItemCreateParams) (*OrderItemInfo, error) {

	var (
		price           float64
		progerssNoodles = types.StepStatusNotStarted
		progressMeat    = types.StepStatusNotStarted
	)

	if params.IncludeNoodles {

		if params.NoodleType == types.NoodleCustom {
			if params.CustomNoodleType == "" {
				return nil, errs.ErrInvalidCustomNoodleType
			} else {
				params.CustomNoodleType = ""
			}
		}

		if params.NoodleType == types.NoodleYi {
			price += float64(params.ExtraNoodleBlocks) * YiNoodlePrice
		}
	} else {

		params.NoodleType = types.NoodleNone
		params.CustomNoodleType = ""
		params.NoodleAmount = types.AdjustmentNone
		params.ExtraNoodleBlocks = 0

		progerssNoodles = types.StepStatusUnrequired
	}

	if len(params.MeatAvailable) == 0 {
		progressMeat = types.StepStatusUnrequired
	}

	if params.Size == types.SizeCustom {
		price += params.CustomSizePrice
	} else {
		price += PriceMap[params.NoodleType.String()][params.Size.String()]
	}

	if params.DiningMethod == types.DiningMethodTakeOut && params.Packaging == types.PackagingPlasticBox {
		price += PlasticBoxPrice
	}

	dailyID, err := l.dailyIDGenerator.NextID(ctx)
	if err != nil {
		return nil, err
	}

	result, err := l.orderItemRepo.Create(ctx, &model.OrderItem{
		DailyID:           uint64(dailyID),
		IncludeNoodles:    params.IncludeNoodles,
		NoodleType:        params.NoodleType,
		CustomNoodleType:  params.CustomNoodleType,
		NoodleAmount:      params.NoodleAmount,
		ExtraNoodleBlocks: params.ExtraNoodleBlocks,
		Size:              params.Size,
		CustomSizePrice:   params.CustomSizePrice,
		MeatAvailable:     params.MeatAvailable,
		MeatExcluded:      params.MeatExcluded,
		Greens:            params.Greens,
		Scallion:          params.Scallion,
		Pepper:            params.Pepper,
		DiningMethod:      params.DiningMethod,
		Packaging:         params.Packaging,
		PackagingMethod:   params.PackagingMethod,
		Note:              params.Note,
		Price:             price,
		ProgressNoodles:   progerssNoodles,
		ProgressMeat:      progressMeat,
	})

	if err != nil {
		return nil, err
	}

	return modelToOrderItemInfo(result), nil

}

// List implements OrderItem.
func (l *OrderItemImpl) List(ctx context.Context, page int, size int) ([]*OrderItemInfo, int64, error) {

	offset := (page - 1) * size

	result, total, err := l.orderItemRepo.List(ctx, offset, size)
	if err != nil {
		return nil, 0, err
	}

	ors := make([]*OrderItemInfo, 0, len(result))
	for _, r := range result {
		ors = append(ors, modelToOrderItemInfo(r))
	}

	return ors, total, nil
}

// UpdateProgress implements OrderItem.
func (l *OrderItemImpl) UpdateProgress(ctx context.Context, params OrderItemUpdateProgressParams) (*OrderItemInfo, error) {
	panic("unimplemented")
}

// UpdateRequires implements OrderItem.
func (l *OrderItemImpl) UpdateRequires(ctx context.Context, params OrderItemUpdateRequiresParams) (*OrderItemInfo, error) {
	panic("unimplemented")
}

func NewOrderItem(dailyIDGenerator *dailyid.DailyIDGenerator, orderItemRepo repo.OrderItem) OrderItem {
	return &OrderItemImpl{
		dailyIDGenerator: dailyIDGenerator,
		orderItemRepo:    orderItemRepo,
	}
}

func modelToOrderItemInfo(m *model.OrderItem) *OrderItemInfo {
	return &OrderItemInfo{
		DailyID:           m.DailyID,
		IncludeNoodles:    m.IncludeNoodles,
		NoodleType:        m.NoodleType,
		CustomNoodleType:  m.CustomNoodleType,
		NoodleAmount:      m.NoodleAmount,
		ExtraNoodleBlocks: m.ExtraNoodleBlocks,
		Size:              m.Size,
		CustomSizePrice:   m.CustomSizePrice,
		MeatAvailable:     m.MeatAvailable,
		MeatExcluded:      m.MeatExcluded,
		Greens:            m.Greens,
		Scallion:          m.Scallion,
		Pepper:            m.Pepper,
		DiningMethod:      m.DiningMethod,
		Packaging:         m.Packaging,
		PackagingMethod:   m.PackagingMethod,
		Note:              m.Note,
		Price:             m.Price,
		ProgressNoodles:   m.ProgressNoodles,
		ProgressMeat:      m.ProgressMeat,
		CompletedAt:       m.CompletedAt,
		CreatedAt:         m.CreatedAt,
	}
}
