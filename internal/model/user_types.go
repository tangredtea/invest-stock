package model

import (
	"errors"
	"strings"
	"time"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	StatusDisabled = 0
	StatusEnabled  = 1
)

var (
	ErrInvalidRole   = errors.New("invalid user role")
	ErrInvalidStatus = errors.New("invalid user status")
	ErrUserNotFound  = errors.New("user not found")
	ErrUserExists    = errors.New("user already exists")
)

// User represents a system user.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func validRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}

func validStatus(status int) bool {
	return status == StatusDisabled || status == StatusEnabled
}

func isDuplicateUserErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint") || strings.Contains(msg, "duplicate")
}
