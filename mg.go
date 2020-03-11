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
		JsonLog            bool              `toml:"json_log"`
		Status             *Status           `toml:"-"`
	}

	Source struct {
		UpSQL   string `json:"-"`
		DownSQL string `json:"-"`
		Apply   bool   `json:"apply"`
		Version uint64 `json:"version"`
		File    string `json:"file"`
	}

	Status struct {
		Error            error
		JsonLog          bool      `json:"-"`
		Section          string    `json:"section"`
		CurrentVersion   uint64    `json:"current"`
		CurrentApplied   *Source   `json:"-"`
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
	m.Status.JsonLog = m.JsonLog
	m.Status.Section = m.Section
	m.Status.CurrentVersion = curVer
	m.Status.BeforeUnapplieds = []*Source{}
	m.Status.AfterUnapplieds = []*Source{}

	diff := make(map[uint64]*Source)
	for _, v := range m.Sources {
		diff[v.Version] = v
	}

	if curVer > 0 {
		rows, err := db.Query(m.VersionSQLBuilder.FetchApplieds())
		if err != nil {
			return err
		}
		for rows.Next() {
			var applied uint64
			if err := rows.Scan(&applied); err != nil {
				return err
			}
			for _, v := range m.Sources {
				if v.Version == applied {
					if applied == curVer {
						m.Status.CurrentApplied = v
					}
					delete(diff, applied)
					break
				}
			}
		}
		defer rows.Close()
		if rows.Err() != nil {
			return err
		}
	}

	for _, v := range diff {
		switch {
		case v.Version < m.Status.CurrentVersion:
			m.Status.BeforeUnapplieds = append(m.Status.BeforeUnapplieds, v)
		case v.Version > m.Status.CurrentVersion:
			m.Status.AfterUnapplieds = append(m.Status.AfterUnapplieds, v)
		}
	}
	sort.Slice(m.Status.BeforeUnapplieds,
		func(i, ii int) bool {
			return m.Status.BeforeUnapplieds[i].Version < m.Status.BeforeUnapplieds[ii].Version
		},
	)
	sort.Slice(m.Status.AfterUnapplieds,
		func(i, ii int) bool {
			return m.Status.AfterUnapplieds[i].Version < m.Status.AfterUnapplieds[ii].Version
		},
	)

	return nil
}

func (m *Migration) Do(do int) error {
	if err := m.do(do); err != nil {
		if m.JsonLog {
			return errors.New(toJson("ERROR", err.Error()))
		}
		return fmt.Errorf("Error: Section is %s %s", m.Section, err.Error())
	}
	return nil
}

func (m *Migration) do(do int) error {
	if err := m.parse(); err != nil {
		return err
	}

	db, err := openDatabase(m.Driver, m.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	var curVer uint64
	if err := db.QueryRow(m.VersionSQLBuilder.FetchCurrentApplied()).Scan(&curVer); err != nil {
		switch do {
		case UpDo, DownDo:
			if _, err := db.Exec(m.VersionSQLBuilder.CreateTable()); err != nil {
				return err
			}
		}
	}
	if curVer < m.VersionStartNumber {
		curVer = m.VersionStartNumber
	}

	if err := m.statusFetch(db, curVer); err != nil {
		return err
	}

	if do == StatusDo {
		d, err := m.Status.display()
		if len(d) > 0 {
			fmt.Print(d)
		}
		return err
	}

	if len(m.Status.sources(do)) == 0 {
		a := fmt.Sprintf("Section %s has no version to migration.", m.Section)
		if m.JsonLog {
			a = (toJson("INFO", a))
		}
		fmt.Println(a)
		return nil
	}

	for _, v := range m.Status.sources(do) {
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
			m.Status.Error = err
			break
		}
		execSQL := fmt.Sprintf("%s%s", mSQL, vSQL)
		if _, err := tx.Exec(execSQL); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				panic(rerr)
			}
			m.Status.Error = err
			break
		}
		if err := tx.Commit(); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				panic(rerr)
			}
			m.Status.Error = err
			break
		}

		v.Apply = true
	}

	d, err := m.Status.displayApplys(do)
	if len(d) > 0 {
		fmt.Println(d)
	}

	return err
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
