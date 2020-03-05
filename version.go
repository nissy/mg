package mg

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type VersionSQLBuilder interface {
	FetchApplieds() string
	FetchCurrentApplied() string
	InsretApplied(version uint64) string
	DeleteApplied(version uint64) string
	CreateTable() string
}

type (
	vPostgres struct {
		table string
	}

	vMySQL struct {
		table string
	}
)

func FetchVersionSQLBuilder(driver, table string) VersionSQLBuilder {
	switch driver {
	case "postgres":
		return &vPostgres{
			table: table,
		}
	case "mysql":
		return &vMySQL{
			table: table,
		}
	}

	return nil
}

func (v *vPostgres) FetchApplieds() string {
	return fmt.Sprintf(
		"SELECT applied_version FROM %s;",
		v.table,
	)
}

func (v *vPostgres) FetchCurrentApplied() string {
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
		"CREATE TABLE IF NOT EXISTS %s (applied_version BIGSERIAL PRIMARY KEY, created_at timestamp with time zone NOT NULL DEFAULT now());",
		v.table,
	)
}

func (v *vMySQL) FetchApplieds() string {
	return fmt.Sprintf(
		"SELECT applied_version FROM %s;",
		v.table,
	)
}

func (v *vMySQL) FetchCurrentApplied() string {
	return fmt.Sprintf(
		"SELECT applied_version FROM %s ORDER BY applied_version DESC LIMIT 1;",
		v.table,
	)
}

func (v *vMySQL) InsretApplied(version uint64) string {
	return fmt.Sprintf(
		"INSERT INTO %s (applied_version) VALUES (%d);",
		v.table, version,
	)
}

func (v *vMySQL) DeleteApplied(version uint64) string {
	return fmt.Sprintf(
		"DELETE FROM %s WHERE applied_version = %d;",
		v.table, version,
	)
}

func (v *vMySQL) CreateTable() string {
	return fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (applied_version bigint(20) unsigned NOT NULL AUTO_INCREMENT, created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (applied_version)) ENGINE=InnoDB;",
		v.table,
	)
}
