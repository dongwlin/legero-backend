package dto

// CreateSessionResponse carries the one-time WebSocket session ticket.
type CreateSessionResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresAt string `json:"expiresAt"`
}
