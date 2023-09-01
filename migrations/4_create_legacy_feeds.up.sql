create table legacy_feeds (
  db_id integer primary key autoincrement,
  content_hash integer not null,
  content text not null,
  etag integer not null,

  user_db_id int not null references users(db_id)
);

create unique index idx_legacy_feeds_user_db
  on legacy_feeds(user_db_id);
