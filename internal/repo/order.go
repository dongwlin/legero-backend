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

	"github.com/dongwlin/legero-backend/internal/model"
)

type Order struct {
	db bun.IDB
}

func NewOrder(db bun.IDB) *Order {
	return &Order{db: db}
}

type listCursor struct {
	Status      model.ListStatus `json:"status"`
	CreatedAt   time.Time        `json:"createdAt"`
	CompletedAt *time.Time       `json:"completedAt,omitempty"`
	ID          uuid.UUID        `json:"id"`
}

func (r *Order) List(ctx context.Context, workspaceID uuid.UUID, query model.ListOrdersQuery) ([]model.Order, *string, error) {
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

	orders := make([]model.Order, 0, limit+1)
	selectQuery := r.db.NewSelect().
		Model(&orders).
		Where("workspace_id = ?", workspaceID)

	switch query.Status {
	case model.ListStatusUncompleted:
		selectQuery = selectQuery.
			Where("completed_at IS NULL").
			OrderExpr("created_at ASC, id ASC")
		if cursor != nil {
			selectQuery = selectQuery.Where("(created_at > ?) OR (created_at = ? AND id > ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
		}
	case model.ListStatusCompleted:
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
	case model.ListStatusAll:
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
	if len(orders) > limit {
		last := orders[limit-1]
		orders = orders[:limit]
		cursorValue, err := encodeCursor(query.Status, last.CreatedAt, last.CompletedAt, last.ID)
		if err != nil {
			return nil, nil, err
		}
		nextCursor = &cursorValue
	}

	return orders, nextCursor, nil
}

func (r *Order) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]model.Order, error) {
	orders := make([]model.Order, 0)
	if err := r.db.NewSelect().
		Model(&orders).
		Where("workspace_id = ?", workspaceID).
		Where("completed_at IS NULL").
		OrderExpr("created_at ASC, id ASC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list active orders: %w", err)
	}
	return orders, nil
}

func (r *Order) GetByID(ctx context.Context, workspaceID, orderID uuid.UUID) (*model.Order, error) {
	order := new(model.Order)
	err := r.db.NewSelect().
		Model(order).
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
	return order, nil
}

func (r *Order) InsertMany(ctx context.Context, orders []model.Order) error {
	if len(orders) == 0 {
		return nil
	}

	if _, err := r.db.NewInsert().Model(&orders).Exec(ctx); err != nil {
		return fmt.Errorf("insert orders: %w", err)
	}
	return nil
}

func (r *Order) Update(ctx context.Context, order *model.Order) error {
	result, err := r.db.NewUpdate().
		Model(order).
		WherePK().
		Where("workspace_id = ?", order.WorkspaceID).
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

func (r *Order) Delete(ctx context.Context, workspaceID, orderID uuid.UUID) (bool, error) {
	result, err := r.db.NewDelete().
		Model((*model.Order)(nil)).
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
		Model((*model.Order)(nil)).
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

func encodeCursor(status model.ListStatus, createdAt time.Time, completedAt *time.Time, id uuid.UUID) (string, error) {
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
