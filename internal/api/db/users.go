package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jdholdren/seymour/internal/seymour"
)

const userNamespace = "-usr"

type User struct {
	ID        string    `db:"id"`
	GithubID  string    `db:"github_id"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r Repo) EnsureUser(ctx context.Context, usr User) (User, error) {
	const q = `INSERT INTO users (id, email, github_id)
	VALUES (:id, :email, :github_id)
	ON CONFLICT (github_id) DO NOTHING;`

	usr.ID = uuid.NewString() + userNamespace
	if _, err := r.db.NamedExecContext(ctx, q, usr); err != nil {
		return User{}, err
	}

	return r.UserByGithubID(ctx, usr.GithubID)
}

func (r Repo) User(ctx context.Context, id string) (User, error) {
	const q = `SELECT * FROM users WHERE id = ?;`

	var usr User
	err := r.db.GetContext(ctx, &usr, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, seymour.ErrNotFound
	}
	if err != nil {
		return User{}, err
	}

	return usr, nil
}

func (r Repo) UserByGithubID(ctx context.Context, githubID string) (User, error) {
	const q = `SELECT * FROM users WHERE github_id = ?;`

	var usr User
	if err := r.db.GetContext(ctx, &usr, q, githubID); err != nil {
		return User{}, err
	}

	return usr, nil
}
