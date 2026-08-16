package repo

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

	"github.com/dongwlin/legero-backend/internal/domain"
	"github.com/dongwlin/legero-backend/internal/repo/schema"
)

type Order struct {
	db bun.IDB
}

func NewOrder(db bun.IDB) *Order {
	return &Order{db: db}
}

type listCursor struct {
	Status      domain.ListStatus `json:"status"`
	CreatedAt   time.Time         `json:"createdAt"`
	CompletedAt *time.Time        `json:"completedAt,omitempty"`
	ID          uuid.UUID         `json:"id"`
}

func (r *Order) List(ctx context.Context, workspaceID uuid.UUID, query domain.ListOrdersQuery) ([]domain.Order, *string, error) {
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

	schemas := make([]schema.Order, 0, limit+1)
	selectQuery := r.db.NewSelect().
		Model(&schemas).
		Where("workspace_id = ?", workspaceID)

	switch query.Status {
	case domain.ListStatusUncompleted:
		selectQuery = selectQuery.
			Where("completed_at IS NULL").
			OrderExpr("created_at ASC, id ASC")
		if cursor != nil {
			selectQuery = selectQuery.Where("(created_at > ?) OR (created_at = ? AND id > ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
		}
	case domain.ListStatusCompleted:
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
	case domain.ListStatusAll:
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
	if len(schemas) > limit {
		last := schemas[limit-1]
		schemas = schemas[:limit]
		cursorValue, err := encodeCursor(query.Status, last.CreatedAt, last.CompletedAt, last.ID)
		if err != nil {
			return nil, nil, err
		}
		nextCursor = &cursorValue
	}

	orders := make([]domain.Order, len(schemas))
	for i := range schemas {
		orders[i] = *toDomainOrder(&schemas[i])
	}
	return orders, nextCursor, nil
}

func (r *Order) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]domain.Order, error) {
	schemas := make([]schema.Order, 0)
	if err := r.db.NewSelect().
		Model(&schemas).
		Where("workspace_id = ?", workspaceID).
		Where("completed_at IS NULL").
		OrderExpr("created_at ASC, id ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list active orders: %w", err)
	}

	orders := make([]domain.Order, len(schemas))
	for i := range schemas {
		orders[i] = *toDomainOrder(&schemas[i])
	}
	return orders, nil
}

func (r *Order) GetByID(ctx context.Context, workspaceID, orderID uuid.UUID) (*domain.Order, error) {
	s := new(schema.Order)
	err := r.db.NewSelect().
		Model(s).
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
	return toDomainOrder(s), nil
}

func (r *Order) InsertMany(ctx context.Context, orders []domain.Order) error {
	if len(orders) == 0 {
		return nil
	}

	schemas := make([]schema.Order, len(orders))
	for i := range orders {
		schemas[i] = *toSchemaOrder(&orders[i])
	}

	if _, err := r.db.NewInsert().Model(&schemas).Exec(ctx); err != nil {
		return fmt.Errorf("insert orders: %w", err)
	}
	return nil
}

// Update persists changes to an order and atomically advances its version.
//
// When expectedVersion is non-nil, the row is updated only if its current
// version matches, which makes the write safe under concurrent updates; a
// mismatch reports domain.ErrOrderConflict. The caller's order.Version is
// refreshed with the version returned by the database.
func (r *Order) Update(ctx context.Context, order *domain.Order, expectedVersion *int64) error {
	s := toSchemaOrder(order)
	query := r.db.NewUpdate().
		Model(s).
		WherePK().
		Where("workspace_id = ?", order.WorkspaceID).
		ExcludeColumn("version").
		Set("version = version + 1").
		Returning("version")
	if expectedVersion != nil {
		query = query.Where("version = ?", *expectedVersion)
	}

	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if expectedVersion != nil {
				return domain.ErrOrderConflict
			}
			return sql.ErrNoRows
		}
		return fmt.Errorf("update order: %w", err)
	}
	order.Version = s.Version
	return nil
}

func (r *Order) Delete(ctx context.Context, workspaceID, orderID uuid.UUID) (bool, error) {
	result, err := r.db.NewDelete().
		Model((*schema.Order)(nil)).
		Where("workspace_id = ?", workspaceID).
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("delete order: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func (r *Order) ClearWorkspace(ctx context.Context, workspaceID uuid.UUID, createdBefore *time.Time) (int, error) {
	query := r.db.NewDelete().
		Model((*schema.Order)(nil)).
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

func toSchemaOrder(o *domain.Order) *schema.Order {
	return &schema.Order{
		ID:                   o.ID,
		WorkspaceID:          o.WorkspaceID,
		DisplayNo:            o.DisplayNo,
		Version:              o.Version,
		StapleTypeCode:       o.StapleTypeCode,
		SizeCode:             o.SizeCode,
		CustomSizePriceCents: o.CustomSizePriceCents,
		StapleAmountCode:     o.StapleAmountCode,
		ExtraStapleUnits:     o.ExtraStapleUnits,
		FriedEggCount:        o.FriedEggCount,
		TofuSkewerCount:      o.TofuSkewerCount,
		SelectedMeatCodes:    o.SelectedMeatCodes,
		GreensCode:           o.GreensCode,
		ScallionCode:         o.ScallionCode,
		PepperCode:           o.PepperCode,
		DiningMethodCode:     o.DiningMethodCode,
		PackagingCode:        o.PackagingCode,
		PackagingMethodCode:  o.PackagingMethodCode,
		TotalPriceCents:      o.TotalPriceCents,
		StapleStepStatusCode: o.StapleStepStatusCode,
		MeatStepStatusCode:   o.MeatStepStatusCode,
		Note:                 o.Note,
		CreatedBy:            o.CreatedBy,
		UpdatedBy:            o.UpdatedBy,
		CreatedAt:            o.CreatedAt,
		UpdatedAt:            o.UpdatedAt,
		CompletedAt:          o.CompletedAt,
	}
}

func toDomainOrder(s *schema.Order) *domain.Order {
	return &domain.Order{
		ID:                   s.ID,
		WorkspaceID:          s.WorkspaceID,
		DisplayNo:            s.DisplayNo,
		Version:              s.Version,
		StapleTypeCode:       s.StapleTypeCode,
		SizeCode:             s.SizeCode,
		CustomSizePriceCents: s.CustomSizePriceCents,
		StapleAmountCode:     s.StapleAmountCode,
		ExtraStapleUnits:     s.ExtraStapleUnits,
		FriedEggCount:        s.FriedEggCount,
		TofuSkewerCount:      s.TofuSkewerCount,
		SelectedMeatCodes:    s.SelectedMeatCodes,
		GreensCode:           s.GreensCode,
		ScallionCode:         s.ScallionCode,
		PepperCode:           s.PepperCode,
		DiningMethodCode:     s.DiningMethodCode,
		PackagingCode:        s.PackagingCode,
		PackagingMethodCode:  s.PackagingMethodCode,
		TotalPriceCents:      s.TotalPriceCents,
		StapleStepStatusCode: s.StapleStepStatusCode,
		MeatStepStatusCode:   s.MeatStepStatusCode,
		Note:                 s.Note,
		CreatedBy:            s.CreatedBy,
		UpdatedBy:            s.UpdatedBy,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
		CompletedAt:          s.CompletedAt,
	}
}

func encodeCursor(status domain.ListStatus, createdAt time.Time, completedAt *time.Time, id uuid.UUID) (string, error) {
	payload := listCursor{
		Status:      status,
		CreatedAt:   createdAt,
		CompletedAt: completedAt,
		ID:          id,
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
		return nil, fmt.Errorf("cursor is invalid: %w", err)
	}

	var cursor listCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, fmt.Errorf("cursor is invalid: %w", err)
	}
	return &cursor, nil
}
