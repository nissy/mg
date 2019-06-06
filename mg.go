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
		Driver       string    `toml:"driver"`
		DSN          string    `toml:"dsn"`
		SourceDir    []string  `toml:"source_dir"`
		Sources      []*Source `toml:"-"`
		VersionTable string    `toml:"version_table"`
		//Transaction  bool      `toml:"transaction"`
	}

	Source struct {
		Up      string
		Down    string
		Path    string
		Version uint64
	}
)

func ReadConfig(filename string) (mg Mg, err error) {
	_, err = toml.DecodeFile(filename, &mg)
	return mg, err
}

func (m *Migration) Up(name string, number int) (err error) {
	return m.run(name, number, false)
}

func (m *Migration) Down(name string, number int) (err error) {
	return m.run(name, number, true)
}

func (m *Migration) run(name string, number int, down bool) (err error) {
	if err := m.parse(); err != nil {
		return err
	}

	vSQL := m.GetVersionSQL()
	if vSQL == nil {
		return errors.New("not driver.")
	}

	db, err := sql.Open(m.Driver, m.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	var applied uint64
	if err := db.QueryRow(vSQL.Fetch()).Scan(&applied); err != nil {
		if _, err := db.Exec(vSQL.CreateTable()); err != nil {
			return err
		}
	}

	var i int
	for _, v := range m.Sources {
		if applied >= v.Version || number > 0 && number <= i {
			continue
		}

		migrateSQL := v.Up
		if down {
			migrateSQL = v.Down
		}
		if _, err := db.Exec(migrateSQL); err != nil {
			return fmt.Errorf("NG %s\nError: %s", v.Path, err.Error())
		}

		fmt.Printf("OK %s\n", v.Path)

		moveSQL := vSQL.Insret(v.Version)
		if down {
			moveSQL = vSQL.Delete(v.Version)
		}
		if _, err := db.Exec(moveSQL); err != nil {
			return err
		}

		i++
	}

	return nil
}

func (m *Migration) parse() (err error) {
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
				return err
			}
		}
	}
	if s.Version == 0 {
		return errors.New("VersionSQL is not found.")
	}

	file, err := os.Open(s.Path)
	if err != nil {
		return err
	}

	defer func() {
		err = file.Close()
	}()

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
				s.Up += line
			}
			if d {
				s.Down += line
			}
		}
		if err == io.EOF {
			break
		}
	}

	return err
}
