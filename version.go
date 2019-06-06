package mg

import (
	"fmt"

	_ "github.com/lib/pq"
)

type VersionSQL interface {
	Fetch() string
	Insret(version uint64) string
	Delete(version uint64) string
	CreateTable() string
}

type VersionPostgres struct {
	table string
}

func (m *Migration) GetVersionSQL() VersionSQL {
	switch m.Driver {
	case "postgres":
		return &VersionPostgres{
			table: m.VersionTable,
		}
	}

	return nil
}

func (v *VersionPostgres) Fetch() string {
	return fmt.Sprintf(
		"SELECT applied_version FROM %s ORDER BY applied_version DESC LIMIT 1",
		v.table,
	)
}

func (v *VersionPostgres) Insret(version uint64) string {
	return fmt.Sprintf(
		"INSERT INTO %s (applied_version) VALUES (%d)",
		v.table, version,
	)
}

func (v *VersionPostgres) Delete(version uint64) string {
	return fmt.Sprintf(
		"DELETE FROM %s WHERE applied_version = %d",
		v.table, version,
	)
}

func (v *VersionPostgres) CreateTable() string {
	return fmt.Sprintf(
		"CREATE TABLE %s (applied_version BIGSERIAL PRIMARY KEY, created_at timestamp with time zone NOT NULL DEFAULT now())",
		v.table,
	)
}
