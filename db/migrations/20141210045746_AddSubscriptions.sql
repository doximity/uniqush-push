
-- +goose Up
create table subscriptions (
  `id` int primary key auto_increment,
  `service_id` int,
  `alias` varchar(40),
  `push_service_provider_type` varchar(10),
  `device_token` varchar(240)
);


-- +goose Down
drop table subscriptions;
