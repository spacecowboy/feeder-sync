create table articles (
  db_id integer primary key autoincrement,
  read_time integer not null,
  identifier text not null,

  user_db_id int not null references users(db_id)
);

create unique index idx_articles_user_db_id_identifier
  on articles(user_db_id, identifier);
create index idx_articles_read_time on articles(read_time);
