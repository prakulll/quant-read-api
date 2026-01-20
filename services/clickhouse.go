package services

import (
	"database/sql"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

var ch *sql.DB

func InitClickHouse(dsn string) error {
	var err error
	ch, err = sql.Open("clickhouse", dsn)
	if err != nil {
		return err
	}
	return ch.Ping()
}

func GetClickHouse() *sql.DB {
	return ch
}
