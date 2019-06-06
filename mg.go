package mg

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	UpToken   = "@migrate.up"
	DownToken = "@migrate.down"
)

type (
	Mg map[string]*Migration

	Migration struct {
		Driver            string            `toml:"driver"`
		DSN               string            `toml:"dsn"`
		SourceDir         []string          `toml:"source_dir"`
		Sources           []*Source         `toml:"-"`
		VersionTable      string            `toml:"version_table"`
		VersionSQLBuilder VersionSQLBuilder `toml:"-"`
	}

	Source struct {
		UpSQL   string
		DownSQL string
		Path    string
		Version uint64
	}
)

func ReadConfig(filename string) (mg Mg, err error) {
	_, err = toml.DecodeFile(filename, &mg)
	return mg, err
}

func (m *Migration) Up(name string) (err error) {
	return m.run(name, false)
}

func (m *Migration) Down(name string) (err error) {
	return m.run(name, true)
}

func (m *Migration) run(name string, down bool) (err error) {
	if err := m.parse(); err != nil {
		return err
	}

	db, err := sql.Open(m.Driver, m.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	var lastVersion uint64
	if err := db.QueryRow(m.VersionSQLBuilder.FetchLastApplied()).Scan(&lastVersion); err != nil {
		if _, err := db.Exec(m.VersionSQLBuilder.CreateTable()); err != nil {
			return err
		}
	}

	for _, v := range m.Sources {
		if down && lastVersion != v.Version {
			continue
		} else if lastVersion >= v.Version {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if down {
			err = v.execDown(tx, m.VersionSQLBuilder)
		} else {
			err = v.execUp(tx, m.VersionSQLBuilder)
		}
		if err != nil {
			tx.Rollback()
			fmt.Printf("NG %s\n", v.Path)
			return fmt.Errorf("Error: %s", err.Error())
		}
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			return err
		}

		fmt.Printf("OK %s\n", v.Path)
	}

	return nil
}

func (s *Source) execUp(tx *sql.Tx, builder VersionSQLBuilder) (err error) {
	if _, err := tx.Exec(s.UpSQL); err != nil {
		return err
	}
	_, err = tx.Exec(builder.InsretApplied(s.Version))

	return err
}

func (s *Source) execDown(tx *sql.Tx, builder VersionSQLBuilder) (err error) {
	if _, err := tx.Exec(s.DownSQL); err != nil {
		return err
	}
	_, err = tx.Exec(builder.DeleteApplied(s.Version))

	return err
}

func (m *Migration) parse() (err error) {
	if m.VersionSQLBuilder = FetchVersionSQLBuilder(m.Driver, m.VersionTable); m.VersionSQLBuilder == nil {
		return errors.New("Error: Driver does not exist.")
	}

	for _, v := range m.SourceDir {
		fs, err := filepath.Glob(filepath.Join(v, "*.sql"))
		if err != nil {
			return err
		}
		for _, vv := range fs {
			s := &Source{
				Path: vv,
			}
			if _, f := filepath.Split(vv); len(f) > 0 {
				s.Version, err = strconv.ParseUint(strings.SplitN(f, "_", 2)[0], 10, 64)
				if err != nil {
					return err
				}
				if err := s.parse(); err != nil {
					return err
				}
				m.Sources = append(m.Sources, s)
			}
		}
	}

	sort.Slice(m.Sources,
		func(i, ii int) bool {
			return m.Sources[i].Version < m.Sources[ii].Version
		},
	)

	return nil
}

func (s *Source) parse() (err error) {
	if _, f := filepath.Split(s.Path); len(f) > 0 {
		if n := strings.SplitN(f, "_", 2); len(n) == 2 {
			if s.Version, err = strconv.ParseUint(strings.SplitN(f, "_", 2)[0], 10, 64); err != nil {
				return fmt.Errorf("Error: Filename is version does not exist %s", s.Path)
			}
		}
	}
	if s.Version == 0 {
		return fmt.Errorf("Error: Filename is version does not exist %s", s.Path)
	}

	file, err := os.Open(s.Path)
	if err != nil {
		return err
	}

	defer file.Close()

	r := bufio.NewReader(file)
	var u, d bool
	for {
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		if strings.HasPrefix(line, "--") {
			if strings.Contains(line, UpToken) {
				u = true
				d = false
			}
			if strings.Contains(line, DownToken) {
				u = false
				d = true
			}
		} else {
			if u {
				s.UpSQL += line
			}
			if d {
				s.DownSQL += line
			}
		}
		if err == io.EOF {
			break
		}
	}

	return err
}
