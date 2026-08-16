alter table orders
  add column version bigint not null default 1;

alter table orders
  add constraint orders_version_positive check (version >= 1);