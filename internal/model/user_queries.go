package model

import (
	"database/sql"
	"errors"
)

const selectUserFields = `SELECT id, username, password, role, status, created_at, updated_at FROM users`

type userRow interface {
	Scan(dest ...any) error
}

func scanUser(row userRow) (*User, error) {
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func ensureUserAffected(res sql.Result) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}
