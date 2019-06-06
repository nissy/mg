package mg

import (
	"fmt"

	_ "github.com/lib/pq"
)

type VersionSQLBuilder interface {
	FetchLastApplied() string
	InsretApplied(version uint64) string
	DeleteApplied(version uint64) string
	CreateTable() string
}

type vPostgres struct {
	table string
}

func FetchVersionSQLBuilder(driver, table string) VersionSQLBuilder {
	switch driver {
	case "postgres":
		return &vPostgres{
			table: table,
		}
	}

	return nil
}

func (v *vPostgres) FetchLastApplied() string {
	return fmt.Sprintf(
		"SELECT applied_version FROM %s ORDER BY applied_version DESC LIMIT 1;",
		v.table,
	)
}

func (v *vPostgres) InsretApplied(version uint64) string {
	return fmt.Sprintf(
		"INSERT INTO %s (applied_version) VALUES (%d);",
		v.table, version,
	)
}

func (v *vPostgres) DeleteApplied(version uint64) string {
	return fmt.Sprintf(
		"DELETE FROM %s WHERE applied_version = %d;",
		v.table, version,
	)
}

func (v *vPostgres) CreateTable() string {
	return fmt.Sprintf(
		"CREATE TABLE %s (applied_version BIGSERIAL PRIMARY KEY, created_at timestamp with time zone NOT NULL DEFAULT now());",
		v.table,
	)
}
