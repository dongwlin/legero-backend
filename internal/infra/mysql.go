package infra

import (
	"context"
	"fmt"

	"github.com/dongwlin/legero-backend/internal/config"
	"github.com/dongwlin/legero-backend/internal/ent"
	_ "github.com/go-sql-driver/mysql"
)

func NewMySQL(conf *config.Config) (*ent.Client, error) {

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=True",
		conf.Database.Username,
		conf.Database.Password,
		conf.Database.Host,
		conf.Database.Port,
		conf.Database.Name,
	)

	db, err := ent.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Schema.Create(context.Background())
	if err != nil {
		return nil, err
	}

	return db, nil
}
