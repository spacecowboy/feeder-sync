-- devices.last_seen
alter table devices
  alter column last_seen drop default;

alter table devices
  alter column last_seen type bigint using extract(epoch from last_seen) * 1000;

alter table devices
  alter column last_seen set default 0;

-- articles.read_time
alter table articles
  alter column read_time drop default;

alter table articles
  alter column read_time type bigint using extract(epoch from read_time) * 1000;

alter table articles
  alter column read_time set default 0;

-- articles.updated_at
alter table articles
  alter column updated_at drop default;

alter table articles
  alter column updated_at type bigint using extract(epoch from updated_at) * 1000;

alter table articles
  alter column updated_at set default 0;
