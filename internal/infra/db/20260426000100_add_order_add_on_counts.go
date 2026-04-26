package db

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, `
alter table orders
  add column if not exists fried_egg_count smallint not null default 0,
  add column if not exists tofu_skewer_count smallint not null default 0;
`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.ExecContext(ctx, `
alter table orders
  drop column if exists tofu_skewer_count,
  drop column if exists fried_egg_count;
`)
		return err
	})
}
