package v1

import "time"

// Viewer is the structured data about the current user in the frontend.
type Viewer struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
