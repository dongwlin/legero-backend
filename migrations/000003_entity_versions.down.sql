alter table refresh_tokens
  drop constraint refresh_tokens_version_positive;

alter table refresh_tokens
  drop column version;

alter table workspaces
  drop constraint workspaces_version_positive;

alter table workspaces
  drop column version;

alter table users
  drop constraint users_version_positive;

alter table users
  drop column version;