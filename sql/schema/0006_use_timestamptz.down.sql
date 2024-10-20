alter table devices
  alter column last_seen type bigint using extract(epoch from last_seen) * 1000;

alter table articles
  alter column read_time type bigint using extract(epoch from read_time) * 1000,
  alter column updated_at type bigint using extract(epoch from updated_at) * 1000;
