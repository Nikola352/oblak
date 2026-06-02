CREATE TABLE invocations
(
    invocation_id   UUID PRIMARY KEY,
    function_id     UUID        NOT NULL REFERENCES functions (function_id),
    status          TEXT        NOT NULL,
    invocation_time TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ DEFAULT NULL
)
