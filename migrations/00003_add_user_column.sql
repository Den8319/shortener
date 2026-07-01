-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS t_urls ADD COLUMN IF NOT EXISTS u_user uuid;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE IF EXISTS t_urls DROP COLUMN IF EXISTS u_user;
-- +goose StatementEnd