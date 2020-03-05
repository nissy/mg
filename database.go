package mg

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func openDatabase(driver, dsn string) (*sql.DB, error) {
	switch driver {
	case "postgres":
	case "mysql":
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return nil, err
		}
		cfg.MultiStatements = true
		cfg.ParseTime = true
		dsn = cfg.FormatDSN()
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
