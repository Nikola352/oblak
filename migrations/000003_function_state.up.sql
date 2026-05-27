CREATE TABLE functions
(
    function_id UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (user_id),
    status      TEXT NOT NULL,
    bucket      TEXT NOT NULL,
    path        TEXT NOT NULL
)
