create table users (
    db_id bigserial primary key,
    user_id text not null,
    legacy_sync_code text not null
);

create unique index idx_users_user_id on users (user_id);
create unique index idx_users_legacy_sync_code on users (legacy_sync_code);
