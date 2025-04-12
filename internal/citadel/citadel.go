// Package fe provides the BFF server for the client side application.
//
// It also handles common functions like user signup and auth in its own state.
package citadel

type user struct {
	ID       string `db:"id"`
	GithubID string `db:"github_id"`
}
