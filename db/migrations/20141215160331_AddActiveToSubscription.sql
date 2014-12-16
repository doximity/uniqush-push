
-- +goose Up
ALTER TABLE subscriptions ADD COLUMN active BOOLEAN DEFAULT TRUE;


-- +goose Down
ALTER TABLE subscriptions DROP COLUMN active;
