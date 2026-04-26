package order

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
)

type Repository interface {
	List(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, query ListOrdersQuery) ([]Order, *string, error)
	ListActive(ctx context.Context, db bun.IDB, workspaceID uuid.UUID) ([]Order, error)
	GetByID(ctx context.Context, db bun.IDB, workspaceID, orderID uuid.UUID) (*Order, error)
	InsertMany(ctx context.Context, db bun.IDB, items []Order) error
	Update(ctx context.Context, db bun.IDB, item Order) error
	Delete(ctx context.Context, db bun.IDB, workspaceID, orderID uuid.UUID) (bool, error)
	ClearWorkspace(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, createdBefore *time.Time) (int, error)
}

type CounterRepository interface {
	Allocate(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, bizDate time.Time, quantity int, now time.Time) (int, error)
	ResetWorkspace(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, bizDateBefore *time.Time) error
}

type BunRepository struct{}

type BunCounterRepository struct{}

type listCursor struct {
	Status      ListStatus `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	ID          uuid.UUID  `json:"id"`
}

func (r *BunRepository) List(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, query ListOrdersQuery) ([]Order, *string, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	cursor, err := decodeCursor(query.Cursor)
	if err != nil {
		return nil, nil, err
	}

	models := make([]OrderModel, 0, limit+1)
	selectQuery := db.NewSelect().
		Model(&models).
		Where("workspace_id = ?", workspaceID)

	switch query.Status {
	case ListStatusUncompleted:
		selectQuery = selectQuery.
			Where("completed_at IS NULL").
			OrderExpr("created_at ASC, id ASC")
		if cursor != nil {
			selectQuery = selectQuery.Where("(created_at > ?) OR (created_at = ? AND id > ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
		}
	case ListStatusCompleted:
		selectQuery = selectQuery.
			Where("completed_at IS NOT NULL").
			OrderExpr("completed_at DESC, created_at DESC, id DESC")
		if cursor != nil && cursor.CompletedAt != nil {
			selectQuery = selectQuery.Where(
				"(completed_at < ?) OR (completed_at = ? AND created_at < ?) OR (completed_at = ? AND created_at = ? AND id < ?)",
				*cursor.CompletedAt,
				*cursor.CompletedAt, cursor.CreatedAt,
				*cursor.CompletedAt, cursor.CreatedAt, cursor.ID,
			)
		}
	case ListStatusAll:
		selectQuery = selectQuery.OrderExpr("created_at DESC, id DESC")
		if cursor != nil {
			selectQuery = selectQuery.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
		}
	default:
		return nil, nil, fmt.Errorf("invalid list status: %s", query.Status)
	}

	if err := selectQuery.Limit(limit + 1).Scan(ctx); err != nil {
		return nil, nil, fmt.Errorf("list orders: %w", err)
	}

	var nextCursor *string
	if len(models) > limit {
		last := models[limit-1]
		models = models[:limit]
		cursorValue, err := encodeCursor(query.Status, last)
		if err != nil {
			return nil, nil, err
		}
		nextCursor = &cursorValue
	}

	return mapOrders(models), nextCursor, nil
}

func (r *BunRepository) ListActive(ctx context.Context, db bun.IDB, workspaceID uuid.UUID) ([]Order, error) {
	var models []OrderModel
	if err := db.NewSelect().
		Model(&models).
		Where("workspace_id = ?", workspaceID).
		Where("completed_at IS NULL").
		OrderExpr("created_at ASC, id ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list active orders: %w", err)
	}
	return mapOrders(models), nil
}

func (r *BunRepository) GetByID(ctx context.Context, db bun.IDB, workspaceID, orderID uuid.UUID) (*Order, error) {
	model := new(OrderModel)
	err := db.NewSelect().
		Model(model).
		Where("workspace_id = ?", workspaceID).
		Where("id = ?", orderID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	item := modelToOrder(*model)
	return &item, nil
}

func (r *BunRepository) InsertMany(ctx context.Context, db bun.IDB, items []Order) error {
	if len(items) == 0 {
		return nil
	}

	models := make([]OrderModel, 0, len(items))
	for _, item := range items {
		models = append(models, orderToModel(item))
	}

	if _, err := db.NewInsert().Model(&models).Exec(ctx); err != nil {
		return fmt.Errorf("insert orders: %w", err)
	}
	return nil
}

func (r *BunRepository) Update(ctx context.Context, db bun.IDB, item Order) error {
	model := orderToModel(item)
	result, err := db.NewUpdate().
		Model(&model).
		WherePK().
		Where("workspace_id = ?", item.WorkspaceID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *BunRepository) Delete(ctx context.Context, db bun.IDB, workspaceID, orderID uuid.UUID) (bool, error) {
	result, err := db.NewDelete().
		Model((*OrderModel)(nil)).
		Where("workspace_id = ?", workspaceID).
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("delete order: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func (r *BunRepository) ClearWorkspace(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, createdBefore *time.Time) (int, error) {
	query := db.NewDelete().
		Model((*OrderModel)(nil)).
		Where("workspace_id = ?", workspaceID)

	if createdBefore != nil {
		query = query.Where("created_at < ?", *createdBefore)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("clear workspace orders: %w", err)
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func (r *BunCounterRepository) Allocate(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, bizDate time.Time, quantity int, now time.Time) (int, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity must be greater than 0")
	}

	if _, err := db.ExecContext(ctx, `
insert into workspace_daily_counters (workspace_id, biz_date, last_seq, updated_at)
values (?, ?, 0, ?)
on conflict (workspace_id, biz_date) do nothing
`, workspaceID, bizDate.Format("2006-01-02"), now); err != nil {
		return 0, fmt.Errorf("ensure workspace counter: %w", err)
	}

	var lastSeq int
	if err := db.NewRaw(`
update workspace_daily_counters
set last_seq = last_seq + ?, updated_at = ?
where workspace_id = ? and biz_date = ?
returning last_seq
`, quantity, now, workspaceID, bizDate.Format("2006-01-02")).Scan(ctx, &lastSeq); err != nil {
		return 0, fmt.Errorf("allocate workspace counter: %w", err)
	}

	return lastSeq - quantity + 1, nil
}

func (r *BunCounterRepository) ResetWorkspace(ctx context.Context, db bun.IDB, workspaceID uuid.UUID, bizDateBefore *time.Time) error {
	query := db.NewDelete().
		Table("workspace_daily_counters").
		Where("workspace_id = ?", workspaceID)

	if bizDateBefore != nil {
		query = query.Where("biz_date < ?", bizDateBefore.Format("2006-01-02"))
	}

	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("reset workspace counter: %w", err)
	}

	return nil
}

func encodeCursor(status ListStatus, model OrderModel) (string, error) {
	payload := listCursor{
		Status:      status,
		CreatedAt:   model.CreatedAt,
		CompletedAt: model.CompletedAt,
		ID:          model.ID,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeCursor(value string) (*listCursor, error) {
	if value == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, httpx.ValidationError("cursor is invalid")
	}

	var cursor listCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, httpx.ValidationError("cursor is invalid")
	}
	return &cursor, nil
}

func mapOrders(models []OrderModel) []Order {
	items := make([]Order, 0, len(models))
	for _, model := range models {
		items = append(items, modelToOrder(model))
	}
	return items
}

func modelToOrder(model OrderModel) Order {
	return Order{
		ID:                   model.ID,
		WorkspaceID:          model.WorkspaceID,
		DisplayNo:            model.DisplayNo,
		StapleTypeCode:       model.StapleTypeCode,
		SizeCode:             model.SizeCode,
		CustomSizePriceCents: model.CustomSizePriceCents,
		StapleAmountCode:     model.StapleAmountCode,
		ExtraStapleUnits:     model.ExtraStapleUnits,
		FriedEggCount:        model.FriedEggCount,
		TofuSkewerCount:      model.TofuSkewerCount,
		SelectedMeatCodes:    cloneInt16s(model.SelectedMeatCodes),
		GreensCode:           model.GreensCode,
		ScallionCode:         model.ScallionCode,
		PepperCode:           model.PepperCode,
		DiningMethodCode:     model.DiningMethodCode,
		PackagingCode:        model.PackagingCode,
		PackagingMethodCode:  model.PackagingMethodCode,
		TotalPriceCents:      model.TotalPriceCents,
		StapleStepStatusCode: model.StapleStepStatusCode,
		MeatStepStatusCode:   model.MeatStepStatusCode,
		Note:                 model.Note,
		CreatedBy:            model.CreatedBy,
		UpdatedBy:            model.UpdatedBy,
		CreatedAt:            model.CreatedAt,
		UpdatedAt:            model.UpdatedAt,
		CompletedAt:          model.CompletedAt,
	}
}

func orderToModel(item Order) OrderModel {
	return OrderModel{
		ID:                   item.ID,
		WorkspaceID:          item.WorkspaceID,
		DisplayNo:            item.DisplayNo,
		StapleTypeCode:       item.StapleTypeCode,
		SizeCode:             item.SizeCode,
		CustomSizePriceCents: item.CustomSizePriceCents,
		StapleAmountCode:     item.StapleAmountCode,
		ExtraStapleUnits:     item.ExtraStapleUnits,
		FriedEggCount:        item.FriedEggCount,
		TofuSkewerCount:      item.TofuSkewerCount,
		SelectedMeatCodes:    cloneInt16s(item.SelectedMeatCodes),
		GreensCode:           item.GreensCode,
		ScallionCode:         item.ScallionCode,
		PepperCode:           item.PepperCode,
		DiningMethodCode:     item.DiningMethodCode,
		PackagingCode:        item.PackagingCode,
		PackagingMethodCode:  item.PackagingMethodCode,
		TotalPriceCents:      item.TotalPriceCents,
		StapleStepStatusCode: item.StapleStepStatusCode,
		MeatStepStatusCode:   item.MeatStepStatusCode,
		Note:                 item.Note,
		CreatedBy:            item.CreatedBy,
		UpdatedBy:            item.UpdatedBy,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
		CompletedAt:          item.CompletedAt,
	}
}
