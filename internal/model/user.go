package model

import (
	"database/sql"
	"fmt"

	"invest/internal/auth"
)

// UserStore provides CRUD operations for users.
type UserStore struct {
	db *sql.DB
}

// NewUserStore creates a UserStore backed by the given DB.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// Create inserts a new user with a hashed password.
func (s *UserStore) Create(username, password, role string) (*User, error) {
	if !validRole(role) {
		return nil, ErrInvalidRole
	}
	hashed, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO users (username, password, role) VALUES (?, ?, ?)`,
		username, hashed, role,
	)
	if err != nil {
		if isDuplicateUserErr(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("insert user id: %w", err)
	}
	return s.GetByID(id)
}

// GetByUsername finds a user by username.
func (s *UserStore) GetByUsername(username string) (*User, error) {
	return scanUser(s.db.QueryRow(selectUserFields+` WHERE username = ?`, username))
}

// GetByID finds a user by ID.
func (s *UserStore) GetByID(id int64) (*User, error) {
	return scanUser(s.db.QueryRow(selectUserFields+` WHERE id = ?`, id))
}

// List returns all users.
func (s *UserStore) List() ([]User, error) {
	rows, err := s.db.Query(selectUserFields + ` ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Update modifies a user's role and status.
func (s *UserStore) Update(id int64, role string, status int) error {
	if !validRole(role) {
		return ErrInvalidRole
	}
	if !validStatus(status) {
		return ErrInvalidStatus
	}
	res, err := s.db.Exec(
		`UPDATE users SET role = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		role, status, id,
	)
	if err != nil {
		return err
	}
	return ensureUserAffected(res)
}

// Delete removes a user by ID.
func (s *UserStore) Delete(id int64) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureUserAffected(res)
}

// SeedAdmin creates the default admin user if no users exist.
func (s *UserStore) SeedAdmin(username, password string) error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("seed admin count: %w", err)
	}
	if count > 0 {
		return nil
	}
	_, err := s.Create(username, password, RoleAdmin)
	return err
}
