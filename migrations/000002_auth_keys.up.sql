CREATE TABLE auth_keys
(
    auth_id    TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (user_id),
    secret_key TEXT NOT NULL
)
