create table users (
  db_id integer primary key autoincrement,
  user_id text not null,
  legacy_sync_code text not null
);

create unique index idx_users_user_id on users(user_id);
create unique index idx_users_legacy_sync_code on users(legacy_sync_code);

create table devices (
  db_id integer primary key autoincrement,
  device_id text not null,
  legacy_device_id int not null,
  device_name text not null,
  last_seen int not null,

  user_db_id int not null references users(db_id)
);

create unique index idx_devices_device_id on devices(device_id);
create unique index idx_devices_user_db_id_legacy_device_id
  on devices(user_db_id, legacy_device_id);
