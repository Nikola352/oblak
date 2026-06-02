package user

import (
	"github.com/google/uuid"
)

type User struct {
	UserId   uuid.UUID `db:"user_id"  json:"user_id"`
	Username string    `db:"username" json:"username"`
	Email    string    `db:"email"    json:"email"`
}
