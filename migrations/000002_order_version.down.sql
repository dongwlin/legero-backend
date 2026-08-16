alter table orders
  drop constraint orders_version_positive;

alter table orders
  drop column version;