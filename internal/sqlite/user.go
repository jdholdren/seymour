package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jdholdren/seymour/internal/seymour"
)

const userNamespace = "-usr"

func (r Repo) EnsureUser(ctx context.Context, usr seymour.User) (seymour.User, error) {
	const q = `INSERT INTO users (id, email, github_id)
	VALUES (:id, :email, :github_id)
	ON CONFLICT (github_id) DO NOTHING;`

	usr.ID = uuid.NewString() + userNamespace
	if _, err := r.db.NamedExecContext(ctx, q, usr); err != nil {
		return seymour.User{}, err
	}

	return r.UserByGithubID(ctx, usr.GithubID)
}

func (r Repo) User(ctx context.Context, id string) (seymour.User, error) {
	const q = `SELECT * FROM users WHERE id = ?;`

	var usr seymour.User
	err := r.db.GetContext(ctx, &usr, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return seymour.User{}, seymour.ErrNotFound
	}
	if err != nil {
		return seymour.User{}, err
	}

	return usr, nil
}

func (r Repo) UserByGithubID(ctx context.Context, githubID string) (seymour.User, error) {
	const q = `SELECT * FROM users WHERE github_id = ?;`

	var usr seymour.User
	if err := r.db.GetContext(ctx, &usr, q, githubID); err != nil {
		return seymour.User{}, err
	}

	return usr, nil
}

// AllUserIDs returns all user IDs from the database.
func (r Repo) AllUserIDs(ctx context.Context) ([]string, error) {
	const q = `SELECT id FROM users;`

	var ids []string
	if err := r.db.SelectContext(ctx, &ids, q); err != nil {
		return nil, err
	}

	return ids, nil
}
