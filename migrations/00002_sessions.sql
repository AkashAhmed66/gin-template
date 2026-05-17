-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_sessions (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    impersonator_id BIGINT NOT NULL DEFAULT 0,
    device_name     VARCHAR(200),
    user_agent      VARCHAR(500),
    ip_address      VARCHAR(64),
    issued_at       TIMESTAMPTZ NOT NULL,
    last_used_at    TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    revoked_reason  VARCHAR(64)
);
CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions (expires_at);
CREATE INDEX idx_user_sessions_revoked_at ON user_sessions (revoked_at);
CREATE INDEX idx_user_sessions_issued_at ON user_sessions (issued_at);
CREATE INDEX idx_user_sessions_impersonator ON user_sessions (impersonator_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_sessions;
-- +goose StatementEnd
