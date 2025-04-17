// Package fe provides the BFF server for the client side application.
//
// It also handles common functions like user signup and auth in its own state.
package citadel

import "time"

type user struct {
	ID        string    `db:"id"`
	GithubID  string    `db:"github_id"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
