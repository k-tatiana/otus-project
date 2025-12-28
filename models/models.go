package models

type Message struct {
	CustomerID string `json:"customer_id"`
	Amount     int    `json:"amount"`
	ReasonID   int    `json:"reason_id"`
}

// User represents an authenticated user in the system.
type User struct {
	ID        int    `json:"id,omitempty"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
