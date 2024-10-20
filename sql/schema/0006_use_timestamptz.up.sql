-- devices.last_seen
alter table devices
  alter column last_seen drop default;

alter table devices
  alter column last_seen type timestamptz using to_timestamp(last_seen / 1000.0);

alter table devices
  alter column last_seen set default now();

-- articles.read_time
alter table articles
  alter column read_time drop default;

alter table articles
  alter column read_time type timestamptz using to_timestamp(read_time / 1000.0);

alter table articles
  alter column read_time set default now();

-- articles.updated_at
alter table articles
  alter column updated_at drop default;

alter table articles
  alter column updated_at type timestamptz using to_timestamp(updated_at / 1000.0);

alter table articles
  alter column updated_at set default now();
