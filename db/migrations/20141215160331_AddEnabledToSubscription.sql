
-- +goose Up
ALTER TABLE subscriptions ADD COLUMN enabled BOOLEAN DEFAULT TRUE;


-- +goose Down
ALTER TABLE subscriptions DROP COLUMN enabled;
