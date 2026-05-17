-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    request_id    VARCHAR(64),
    timestamp     TIMESTAMPTZ NOT NULL,
    duration_ms   BIGINT NOT NULL DEFAULT 0,
    user_id       BIGINT,
    username      VARCHAR(100),
    method        VARCHAR(10) NOT NULL,
    path          VARCHAR(500) NOT NULL,
    query_string  VARCHAR(1000),
    status_code   INTEGER NOT NULL,
    action        VARCHAR(100),
    resource_type VARCHAR(100),
    resource_id   VARCHAR(100),
    client_ip     VARCHAR(64),
    user_agent    VARCHAR(500),
    request_body  TEXT,
    response_body TEXT,
    error_message TEXT
);
CREATE INDEX idx_audit_logs_request_id ON audit_logs (request_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs (timestamp);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_username ON audit_logs (username);
CREATE INDEX idx_audit_logs_method ON audit_logs (method);
CREATE INDEX idx_audit_logs_path ON audit_logs (path);
CREATE INDEX idx_audit_logs_action ON audit_logs (action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs (resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs (resource_id);
CREATE INDEX idx_audit_logs_status_code ON audit_logs (status_code);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_logs;
-- +goose StatementEnd
