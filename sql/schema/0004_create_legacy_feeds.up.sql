create table legacy_feeds (
  -- db_id integer primary key autoincrement,
  db_id bigserial primary key,
  content_hash bigint not null,
  content text not null,
  etag text not null,

  user_db_id bigint not null references users(db_id)
);

create unique index idx_legacy_feeds_user_db
  on legacy_feeds(user_db_id);
