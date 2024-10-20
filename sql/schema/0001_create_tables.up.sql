create table users (
  -- db_id integer primary key autoincrement,
  db_id bigserial primary key,
  user_id text not null,
  legacy_sync_code text not null
);

create unique index idx_users_user_id on users(user_id);
create unique index idx_users_legacy_sync_code on users(legacy_sync_code);

create table devices (
  -- db_id integer primary key autoincrement,
  db_id bigserial primary key,
  device_id text not null,
  legacy_device_id bigint not null,
  device_name text not null,
  last_seen bigint not null,

  user_db_id bigint not null references users(db_id)
);

create unique index idx_devices_device_id on devices(device_id);
create unique index idx_devices_user_db_id_legacy_device_id
  on devices(user_db_id, legacy_device_id);
