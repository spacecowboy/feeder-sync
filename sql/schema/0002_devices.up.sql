create table devices (
    db_id bigserial primary key,
    device_id text not null,
    device_name text not null,
    last_seen timestamptz not null,
    legacy_device_id bigint not null,

    user_db_id bigint not null references users (db_id)
);

create unique index idx_devices_device_id on devices (device_id);
create unique index idx_devices_user_db_id_legacy_device_id
on devices (user_db_id, legacy_device_id);
