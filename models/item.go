package models

// Item represents a simple item stored in memory.
type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// PointsInfo represents loyalty points information.
type PointsInfo struct {
	UserID  string `json:"user_id"`
	Balance int    `json:"balance"`
	Level   string `json:"level"`
}
