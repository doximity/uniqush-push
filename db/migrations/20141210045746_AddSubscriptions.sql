
-- +goose Up
create table subscriptions (
  `id` int primary key auto_increment,
  `service_id` int,
  `alias` varchar(40),
  `push_service_provider_type` varchar(10),
  `device_key` varchar(240) NOT NULL
);


-- +goose Down
drop table subscriptions;
