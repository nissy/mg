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

	"github.com/nissy/envexpand/toml"
)

const (
	UpDo = iota
	DownDo
	StatusDo
)

var (
	DefaultUpAnnotation   = "@migrate.up"
	DefaultDownAnnotation = "@migrate.down"
	DefaultVersionTable   = "migration_versions"
)

type (
	Mg map[string]*Migration

	Migration struct {
		Section            string            `toml:"-"`
		Driver             string            `toml:"driver"`
		DSN                string            `toml:"dsn"`
		SourceDir          []string          `toml:"source_dir"`
		Sources            []*Source         `toml:"-"`
		VersionTable       string            `toml:"version_table"`
		VersionStartNumber uint64            `toml:"version_start_number"`
		VersionSQLBuilder  VersionSQLBuilder `toml:"-"`
		UpAnnotation       string            `toml:"up_annotation"`
		DownAnnotation     string            `toml:"down_annotation"`
		Apply              bool              `toml:"-"`
		JsonLog            bool              `toml:"json_log"`
		Status             *Status           `toml:"-"`
	}

	Source struct {
		UpSQL   string `json:"-"`
		DownSQL string `json:"-"`
		File    string `json:"file"`
		Version uint64 `json:"version"`
		Apply   bool   `json:"apply"`
	}

	Status struct {
		Current          uint64    `json:"current"`
		BeforeUnapplieds []*Source `json:"before_unapplieds"`
		AfterUnapplieds  []*Source `json:"after_unapplieds"`
	}
)

func OpenCfg(filename string) (mg Mg, err error) {
	if err = toml.Open(filename, &mg); err != nil {
		return nil, err
	}
	for s, m := range mg {
		if err := m.init(s); err != nil {
			return nil, err
		}
	}

	return mg, nil
}

func (m *Migration) init(section string) error {
	if len(section) == 0 {
		return errors.New("Section name does not exist.")
	}
	m.Section = section
	m.Status = &Status{}

	if len(m.VersionTable) == 0 {
		m.VersionTable = DefaultVersionTable
	}
	if len(m.UpAnnotation) == 0 {
		m.UpAnnotation = DefaultUpAnnotation
	}
	if len(m.DownAnnotation) == 0 {
		m.DownAnnotation = DefaultDownAnnotation
	}
	if m.VersionSQLBuilder = FetchVersionSQLBuilder(m.Driver, m.VersionTable); m.VersionSQLBuilder == nil {
		return fmt.Errorf("Driver is %s does not exist.", m.Driver)
	}

	return nil
}

func (m *Migration) statusFetch(db *sql.DB, curVer uint64) error {
	m.Status.Current = curVer
	rows, err := db.Query(m.VersionSQLBuilder.FetchApplieds())
	if err != nil {
		return err
	}
	diff := make(map[uint64]*Source)
	for _, v := range m.Sources {
		diff[v.Version] = v
	}
	for rows.Next() {
		var applied uint64
		if err := rows.Scan(&applied); err != nil {
			return err
		}
		for _, v := range m.Sources {
			if v.Version == applied {
				delete(diff, applied)
				break
			}
		}
	}
	defer rows.Close()
	if rows.Err() != nil {
		return err
	}
	for _, v := range diff {
		switch {
		case v.Version < m.Status.Current:
			m.Status.BeforeUnapplieds = append(m.Status.BeforeUnapplieds, v)
		case v.Version > m.Status.Current:
			m.Status.AfterUnapplieds = append(m.Status.AfterUnapplieds, v)
		}
	}

	return nil
}

func (m *Migration) Do(do int) (err error) {
	if err := m.parse(); err != nil {
		return err
	}

	db, err := openDatabase(m.Driver, m.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	logger := m.NewLogging()

	var curVer uint64
	if err := db.QueryRow(m.VersionSQLBuilder.FetchCurrentApplied()).Scan(&curVer); err != nil {
		switch do {
		case UpDo, DownDo:
			if _, err := db.Exec(m.VersionSQLBuilder.CreateTable()); err != nil {
				return err
			}
		case StatusDo:
			fmt.Println(err.Error())
		}
	}
	if curVer < m.VersionStartNumber {
		curVer = m.VersionStartNumber
	}

	if err := m.statusFetch(db, curVer); err != nil {
		return err
	}

	if do == StatusDo {
		fmt.Print(logger.status(m.Status))
		if len(m.Status.BeforeUnapplieds) > 0 {
			err = errors.New("Unapplied version exists before current version.")
		}
		return err
	}

	for _, v := range m.Status.AfterUnapplieds {
		var mSQL, vSQL string
		switch do {
		case UpDo:
			mSQL = v.UpSQL
			vSQL = m.VersionSQLBuilder.InsretApplied(v.Version)
		case DownDo:
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

		execSQL := fmt.Sprintf("%s%s", mSQL, vSQL)
		if _, err := tx.Exec(execSQL); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				panic(rerr)
			}
			fmt.Println(logger.source(v))
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
		fmt.Println(logger.source(v))
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
				File: vv,
			}
			if _, f := filepath.Split(vv); len(f) > 0 {
				if err := s.parse(m.UpAnnotation, m.DownAnnotation); err != nil {
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

func (s *Source) parse(upAnot, downAnot string) (err error) {
	if _, f := filepath.Split(s.File); len(f) > 0 {
		if s.Version, err = nameToVersion(f); err != nil {
			return err
		}
	}
	if s.Version == 0 {
		return fmt.Errorf("Filename is version does not exist %s", s.File)
	}

	file, err := os.Open(s.File)
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
			if strings.Contains(line, upAnot) {
				u = true
				d = false
			}
			if strings.Contains(line, downAnot) {
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

func nameToVersion(name string) (version uint64, err error) {
	i := 0
	for _, v := range name {
		if v >= 48 && v <= 57 {
			i++
			continue
		}
		break
	}
	if i > 0 {
		version, err = strconv.ParseUint(name[0:i], 10, 64)
	}
	if err != nil || version == 0 {
		return 0, fmt.Errorf("Filename is version does not exist %s", name)
	}

	return version, nil
}
