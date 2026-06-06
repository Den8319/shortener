-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS t_urls ADD COLUMN IF NOT EXISTS b_deleted boolean;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE IF EXISTS t_urls DROP COLUMN IF EXISTS b_deleted;
-- +goose StatementEnd
