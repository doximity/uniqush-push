
-- +goose Up
ALTER TABLE subscriptions CHANGE device_key subscription_key VARCHAR(240) NOT NULL;
ALTER TABLE subscriptions ADD COLUMN device_key VARCHAR(100) NOT NULL AFTER service_id;


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE subscriptions DROP COLUMN device_key;
ALTER TABLE subscriptions CHANGE subscription_key device_key VARCHAR(240);

