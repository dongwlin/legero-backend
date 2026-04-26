package order

import "github.com/google/uuid"

const (
	EventOrderUpsert  = "order.upsert"
	EventOrderDeleted = "order.deleted"
	EventOrderCleared = "order.cleared"
)

type Publisher interface {
	Publish(workspaceID uuid.UUID, eventType string, payload any)
}

type UpsertEvent struct {
	Item OrderDTO `json:"item"`
}

type DeletedEvent struct {
	ID string `json:"id"`
}

type ClearedEvent struct {
	ClearedCount int                `json:"clearedCount"`
	Mode         ClearWorkspaceMode `json:"mode"`
}
