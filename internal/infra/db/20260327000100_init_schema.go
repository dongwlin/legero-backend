package db

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, `
create table if not exists users (
  id uuid primary key,
  phone text not null unique,
  password_hash text not null,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists workspaces (
  id uuid primary key,
  name text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists workspace_members (
  workspace_id uuid not null references workspaces(id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  role text not null check (role in ('owner', 'staff')),
  created_at timestamptz not null default now(),
  primary key (workspace_id, user_id)
);

create index if not exists workspace_members_user_id_idx
on workspace_members(user_id);

create table if not exists refresh_tokens (
  id uuid primary key,
  user_id uuid not null references users(id) on delete cascade,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  token_hash text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  rotated_at timestamptz,
  revoked_at timestamptz,
  replaced_by_id uuid references refresh_tokens(id)
);

create index if not exists refresh_tokens_user_id_idx
on refresh_tokens(user_id);

create index if not exists refresh_tokens_expires_at_idx
on refresh_tokens(expires_at);

create table if not exists workspace_daily_counters (
  workspace_id uuid not null references workspaces(id) on delete cascade,
  biz_date date not null,
  last_seq integer not null,
  updated_at timestamptz not null default now(),
  primary key (workspace_id, biz_date)
);

create table if not exists orders (
  id uuid primary key,
  workspace_id uuid not null references workspaces(id) on delete cascade,
  display_no text not null,

  staple_type_code smallint,
  size_code smallint not null,
  custom_size_price_cents integer,
  staple_amount_code smallint not null,
  extra_staple_units smallint not null default 0,

  selected_meat_codes smallint[] not null default '{}',

  greens_code smallint not null,
  scallion_code smallint not null,
  pepper_code smallint not null,

  dining_method_code smallint not null,
  packaging_code smallint,
  packaging_method_code smallint,

  total_price_cents integer not null,
  staple_step_status_code smallint not null,
  meat_step_status_code smallint not null,

  note text not null default '',

  created_by uuid not null references users(id),
  updated_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  completed_at timestamptz
);

create index if not exists orders_workspace_created_at_idx
on orders(workspace_id, created_at desc);

create index if not exists orders_workspace_completed_at_idx
on orders(workspace_id, completed_at);

create index if not exists orders_workspace_active_idx
on orders(workspace_id, created_at desc)
where completed_at is null;

create unique index if not exists orders_workspace_display_no_idx
on orders(workspace_id, display_no);
`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, `
drop table if exists orders;
drop table if exists workspace_daily_counters;
drop table if exists refresh_tokens;
drop table if exists workspace_members;
drop table if exists workspaces;
drop table if exists users;
`)
		return err
	})
}
