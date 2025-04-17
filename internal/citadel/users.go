package citadel

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type userRepo struct {
	db *sqlx.DB
}

const userNamespace = "-usr"

func (ur userRepo) ensureUser(ctx context.Context, usr user) (user, error) {
	const q = `INSERT INTO users (id, email, github_id)
	VALUES (:id, :email, :github_id)
	ON CONFLICT (github_id) DO NOTHING;`

	usr.ID = uuid.NewString() + userNamespace
	if _, err := ur.db.NamedExecContext(ctx, q, usr); err != nil {
		return user{}, err
	}

	return ur.userByGithubID(ctx, usr.GithubID)
}

func (ur userRepo) user(ctx context.Context, id string) (user, error) {
	const q = `SELECT * FROM users WHERE id = ?;`

	var usr user
	if err := ur.db.GetContext(ctx, &usr, q, id); err != nil {
		return user{}, err
	}

	return usr, nil
}

func (ur userRepo) userByGithubID(ctx context.Context, githubID string) (user, error) {
	const q = `SELECT * FROM users WHERE github_id = ?;`

	var usr user
	if err := ur.db.GetContext(ctx, &usr, q, githubID); err != nil {
		return user{}, err
	}

	return usr, nil
}
