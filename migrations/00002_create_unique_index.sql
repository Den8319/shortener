-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE  INDEX IF NOT EXISTS idx_data_long ON t_urls(s_long_url);
CREATE UNIQUE  INDEX IF NOT EXISTS idx_data_short ON t_urls(s_short_url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_data_long;
DROP INDEX IF EXISTS idx_data_short;
-- +goose StatementEnd