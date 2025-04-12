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

func (ur userRepo) ensureUser(ctx context.Context, usr user) error {
	const q = `INSERT INTO users (id, email, github_id)
	VALUES (:id, :email, :github_id)
	ON CONFLICT (github_id) DO NOTHING;`

	usr.ID = uuid.NewString() + userNamespace
	if _, err := ur.db.NamedExecContext(ctx, q, usr); err != nil {
		return err
	}

	return nil
}
