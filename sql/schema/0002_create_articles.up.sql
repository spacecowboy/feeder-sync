create table articles (
  -- db_id integer primary key autoincrement,
  db_id bigserial primary key,
  read_time bigint not null,
  identifier text not null,

  user_db_id bigint not null references users(db_id)
);

create unique index idx_articles_user_db_id_identifier
  on articles(user_db_id, identifier);
create index idx_articles_read_time on articles(read_time);
