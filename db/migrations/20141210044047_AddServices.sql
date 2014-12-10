
-- +goose Up
create table services (
  `id` int primary key auto_increment,
  `alias` varchar(100)
);

create table push_service_providers (
  `id` int primary key auto_increment,
  `service_id` int,
  `type` varchar(10)
);

create table apns_access_keys (
  `id` int primary key auto_increment,
  `push_service_provider_id` int,
  `certficate` text,
  `key` text
);

create table gcm_access_keys (
  `id` int primary key auto_increment,
  `push_service_provider_id` int,
  `project` varchar(240),
  `key` varchar(240)
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
drop table services;
drop table push_service_providers;
drop table apns_access_keys;
drop table gcm_access_keys;
