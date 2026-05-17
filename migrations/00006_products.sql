-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    sku         VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    price       DOUBLE PRECISION NOT NULL DEFAULT 0,
    stock       INTEGER NOT NULL DEFAULT 0,
    image_url   VARCHAR(500),
    status      VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(100),
    updated_by  VARCHAR(100),
    deleted_at  TIMESTAMPTZ,
    deleted_by  VARCHAR(100),
    version     BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX idx_products_name ON products (name);
CREATE INDEX idx_products_status ON products (status);
CREATE INDEX idx_products_deleted_at ON products (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS products;
-- +goose StatementEnd
