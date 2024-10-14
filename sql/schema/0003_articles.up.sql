create table articles (
    db_id bigserial primary key,
    read_time timestamptz not null,
    identifier text not null,
    updated_at timestamptz not null,

    user_db_id bigint not null references users (db_id)
);

create unique index idx_articles_user_db_id_identifier
on articles (user_db_id, identifier);
create index idx_articles_read_time on articles (read_time);
create index idx_articles_updated_at on articles (updated_at);
