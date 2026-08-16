alter table users
  add column version bigint not null default 1;

alter table users
  add constraint users_version_positive check (version >= 1);

alter table workspaces
  add column version bigint not null default 1;

alter table workspaces
  add constraint workspaces_version_positive check (version >= 1);

alter table refresh_tokens
  add column version bigint not null default 1;

alter table refresh_tokens
  add constraint refresh_tokens_version_positive check (version >= 1);