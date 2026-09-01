package dto

// CreateSessionResponseData carries the one-time WebSocket session ticket.
type CreateSessionResponseData struct {
	Ticket    string `json:"ticket"`
	ExpiresAt string `json:"expiresAt"`
}
