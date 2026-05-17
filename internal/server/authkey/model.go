package authkey

import "github.com/google/uuid"

type AuthKey struct {
	AuthId    string    `db:"auth_id"`
	UserId    uuid.UUID `db:"user_id"`
	SecretKey string    `db:"secret_key"`
}
