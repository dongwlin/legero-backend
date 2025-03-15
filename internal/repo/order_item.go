package repo

import (
	"context"

	"github.com/dongwlin/legero-backend/internal/ent"
	"github.com/dongwlin/legero-backend/internal/ent/orderitem"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/dongwlin/legero-backend/internal/model/types"
)

type OrderItem interface {
	Create(ctx context.Context, params *model.OrderItem) (*model.OrderItem, error)
	Update(ctx context.Context, params *model.OrderItem) (*model.OrderItem, error)
	List(ctx context.Context, limit, offset int) ([]*model.OrderItem, int64, error)
}

type OrderItemImpl struct {
	db *ent.Client
}

// Create implements OrderItem.
func (r *OrderItemImpl) Create(ctx context.Context, params *model.OrderItem) (*model.OrderItem, error) {

	result, err := r.db.OrderItem.Create().
		SetDailyID(params.DailyID).
		SetIncludeNoodles(params.IncludeNoodles).
		SetNoodleType(orderitem.NoodleType(params.NoodleType)).
		SetCustomNoodleType(params.CustomNoodleType).
		SetNoodleAmount(orderitem.NoodleAmount(params.NoodleAmount)).
		SetExtraNoodleBlocks(params.ExtraNoodleBlocks).
		SetSize(orderitem.Size(params.Size)).
		SetCustomSizePrice(params.CustomSizePrice).
		SetMeatAvailable(types.MeatsToStrings(params.MeatAvailable)).
		SetMeatExcluded(types.MeatsToStrings(params.MeatExcluded)).
		SetGreens(orderitem.Greens(params.Greens)).
		SetScallion(orderitem.Scallion(params.Scallion)).
		SetPepper(orderitem.Pepper(params.Pepper)).
		SetDiningMethod(orderitem.DiningMethod(params.DiningMethod)).
		SetPackaging(orderitem.Packaging(params.Packaging)).
		SetPackagingMethod(orderitem.PackagingMethod(params.PackagingMethod)).
		SetNote(params.Note).
		SetPrice(params.Price).
		SetProgressNoodles(orderitem.ProgressNoodles(params.ProgressNoodles)).
		SetProgressMeat(orderitem.ProgressMeat(params.ProgressMeat)).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return model.EntToOrderItem(result), nil
}

// List implements OrderItem.
func (r *OrderItemImpl) List(ctx context.Context, limit int, offset int) ([]*model.OrderItem, int64, error) {

	query := r.db.OrderItem.Query()

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	result, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	ors := make([]*model.OrderItem, 0, len(result))
	for _, r := range result {
		ors = append(ors, model.EntToOrderItem(r))
	}

	return ors, int64(total), nil
}

// Update implements OrderItem.
func (r *OrderItemImpl) Update(ctx context.Context, params *model.OrderItem) (*model.OrderItem, error) {

	result, err := r.db.OrderItem.UpdateOneID(params.ID).
		SetIncludeNoodles(params.IncludeNoodles).
		SetNoodleType(orderitem.NoodleType(params.NoodleType)).
		SetCustomNoodleType(params.CustomNoodleType).
		SetNoodleAmount(orderitem.NoodleAmount(params.NoodleAmount)).
		SetExtraNoodleBlocks(params.ExtraNoodleBlocks).
		SetSize(orderitem.Size(params.Size)).
		SetCustomSizePrice(params.CustomSizePrice).
		SetMeatAvailable(types.MeatsToStrings(params.MeatAvailable)).
		SetMeatExcluded(types.MeatsToStrings(params.MeatExcluded)).
		SetGreens(orderitem.Greens(params.Greens)).
		SetScallion(orderitem.Scallion(params.Scallion)).
		SetPepper(orderitem.Pepper(params.Pepper)).
		SetDiningMethod(orderitem.DiningMethod(params.DiningMethod)).
		SetPackaging(orderitem.Packaging(params.Packaging)).
		SetPackagingMethod(orderitem.PackagingMethod(params.PackagingMethod)).
		SetNote(params.Note).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return model.EntToOrderItem(result), nil
}

func NewOrderItem(db *ent.Client) OrderItem {
	return &OrderItemImpl{
		db: db,
	}
}
