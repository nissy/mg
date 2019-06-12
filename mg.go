package mg

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	UpDo = iota
	DownDo

	UpToken   = "@migrate.up"
	DownToken = "@migrate.down"
)

type (
	Mg map[string]*Migration

	Migration struct {
		Section           string            `toml:"-"`
		Driver            string            `toml:"driver"`
		DSN               string            `toml:"dsn"`
		SourceDir         []string          `toml:"source_dir"`
		Sources           []*Source         `toml:"-"`
		VersionTable      string            `toml:"version_table"`
		VersionSQLBuilder VersionSQLBuilder `toml:"-"`
		UpToken           string            `toml:"up_token"`
		DownToken         string            `toml:"down_token"`
		Apply             bool              `toml:"-"`
	}

	Source struct {
		UpSQL   string
		DownSQL string
		Path    string
		Version uint64
		Apply   bool
	}
)

func ReadConfig(filename string) (mg Mg, err error) {
	if _, err = toml.DecodeFile(filename, &mg); err != nil {
		return nil, err
	}
	for s, m := range mg {
		m.Section = s
		if m.VersionSQLBuilder = FetchVersionSQLBuilder(m.Driver, m.VersionTable); m.VersionSQLBuilder == nil {
			return nil, fmt.Errorf("Driver is %s does not exist.", m.Driver)
		}
		if len(m.UpToken) == 0 {
			m.UpToken = UpToken
		}
		if len(m.DownToken) == 0 {
			m.DownToken = DownToken
		}
	}

	return mg, err
}

func (m *Migration) Status() (err error) {
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
		fmt.Println(err.Error())
	}
	fmt.Printf("Version status to %s:\n\tcurrent: %d\n", m.Section, lastVersion)
	for _, v := range m.Sources {
		if lastVersion >= v.Version {
			continue
		}
		fmt.Printf("\t\x1b[31munapplied: %d %s\x1b[0m\n", v.Version, v.Path)
	}

	return nil
}

func (m *Migration) Exec(do int) (err error) {
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
		var mSQL, vSQL string
		switch do {
		case UpDo:
			if lastVersion >= v.Version {
				continue
			}
			mSQL = v.UpSQL
			vSQL = m.VersionSQLBuilder.InsretApplied(v.Version)
		case DownDo:
			if lastVersion != v.Version {
				continue
			}
			mSQL = v.DownSQL
			vSQL = m.VersionSQLBuilder.DeleteApplied(v.Version)
		}
		if len(mSQL) == 0 {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(fmt.Sprintf("%s%s", mSQL, vSQL)); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				panic(rerr)
			}
			fmt.Printf("\x1b[31mNG %s\x1b[0m\n", v.Path)
			return err
		}
		if err := tx.Commit(); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				panic(rerr)
			}
			return err
		}

		v.Apply = true
		m.Apply = true
		fmt.Printf("OK %s\n", v.Path)
	}

	if !m.Apply {
		fmt.Printf("%s has no version to migration.\n", m.Section)
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
				if err := s.parse(m.UpToken, m.DownToken); err != nil {
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

func (s *Source) parse(tokenUp, tokenDown string) (err error) {
	if _, f := filepath.Split(s.Path); len(f) > 0 {
		if s.Version, err = fileNameToVersion(f); err != nil {
			return err
		}
	}
	if s.Version == 0 {
		return fmt.Errorf("Filename is version does not exist %s", s.Path)
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
			if strings.Contains(line, tokenUp) {
				u = true
				d = false
			}
			if strings.Contains(line, tokenDown) {
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

func fileNameToVersion(filename string) (version uint64, err error) {
	i := 0
	for _, v := range filename {
		if v >= 48 && v <= 57 {
			i++
			continue
		}
		break
	}
	if i > 0 {
		version, err = strconv.ParseUint(filename[0:i], 10, 64)
	}
	if err != nil || version == 0 {
		return 0, fmt.Errorf("Filename is version does not exist %s", filename)
	}

	return version, nil
}
