package seymour

import (
	"context"
	"time"
)

type UserService interface {
	EnsureUser(ctx context.Context, usr User) (User, error)
	User(ctx context.Context, id string) (User, error)
	UserByGithubID(ctx context.Context, githubID string) (User, error)
	AllUserIDs(ctx context.Context) ([]string, error)
}

type User struct {
	ID        string    `db:"id"`
	GithubID  string    `db:"github_id"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
