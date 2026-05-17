-- +goose Up
-- +goose StatementBegin
CREATE TABLE idempotency_records (
    id            BIGSERIAL PRIMARY KEY,
    key           VARCHAR(200) NOT NULL,
    method        VARCHAR(10) NOT NULL,
    path          VARCHAR(500) NOT NULL,
    user_id       BIGINT NOT NULL,
    request_hash  VARCHAR(128) NOT NULL,
    status_code   INTEGER NOT NULL,
    response_body BYTEA,
    created_at    TIMESTAMPTZ NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    UNIQUE (key, method, path, user_id)
);
CREATE INDEX idx_idem_created_at ON idempotency_records (created_at);
CREATE INDEX idx_idem_expires_at ON idempotency_records (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS idempotency_records;
-- +goose StatementEnd
