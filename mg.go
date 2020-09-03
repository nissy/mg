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
	UpDo      = "up"
	DownDo    = "down"
	ForceUpDo = "force-up"
	StatusDo  = "status"
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
		JsonFormat         bool              `toml:"json_format"`
		status             *status           `toml:"-"`
	}

	Source struct {
		UpSQL     string `json:"-"`
		DownSQL   string `json:"-"`
		Apply     bool   `json:"apply"`
		Version   uint64 `json:"version"`
		File      string `json:"file"`
		Duplicate bool   `json:"duplicate"`
	}

	status struct {
		Error            error
		CurrentVersion   uint64
		CurrentSource    *Source
		UnappliedSources []*Source
		ApplySources     []*Source
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
		if err := m.parse(); err != nil {
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
	m.status = &status{
		ApplySources:     []*Source{},
		UnappliedSources: []*Source{},
	}

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

func (m *Migration) fetchApplied(db *sql.DB, do string) error {
	if err := db.QueryRow(m.VersionSQLBuilder.FetchCurrentApplied()).Scan(&m.status.CurrentVersion); err != nil {
		switch do {
		case UpDo, ForceUpDo, DownDo:
			if _, err := db.Exec(m.VersionSQLBuilder.CreateTable()); err != nil {
				return err
			}
		}
	}
	if m.status.CurrentVersion < m.VersionStartNumber {
		m.status.CurrentVersion = m.VersionStartNumber
	}

	diff := make(map[uint64]*Source)
	for _, v := range m.Sources {
		if m.VersionStartNumber < v.Version {
			//duplicate
			if _, ok := diff[v.Version]; ok {
				v.Duplicate = true
				m.status.UnappliedSources = append(m.status.UnappliedSources, v)
				continue
			}
			diff[v.Version] = v
		}
	}
	if m.status.CurrentVersion > 0 {
		rows, existErr := db.Query(m.VersionSQLBuilder.FetchApplied())
		if existErr != nil {
			if do != StatusDo {
				return existErr
			}
		} else {
			for rows.Next() {
				var applied uint64
				if err := rows.Scan(&applied); err != nil {
					return err
				}
				for _, v := range m.Sources {
					if v.Version == applied {
						if applied == m.status.CurrentVersion {
							m.status.CurrentSource = v
						}
						delete(diff, applied)
						break
					}
				}
			}
			defer rows.Close()
			if rows.Err() != nil {
				return rows.Err()
			}
		}
	}

	for _, v := range diff {
		switch {
		case v.Version < m.status.CurrentVersion:
			m.status.UnappliedSources = append(m.status.UnappliedSources, v)
		case v.Version > m.status.CurrentVersion:
			m.status.ApplySources = append(m.status.ApplySources, v)
		}
	}
	sort.Slice(m.status.UnappliedSources,
		func(i, ii int) bool {
			return m.status.UnappliedSources[i].Version < m.status.UnappliedSources[ii].Version
		},
	)
	sort.Slice(m.status.ApplySources,
		func(i, ii int) bool {
			return m.status.ApplySources[i].Version < m.status.ApplySources[ii].Version
		},
	)

	return nil
}

func (m *Migration) Do(do string) error {
	if err := m.do(do); err != nil {
		if m.JsonFormat {
			var e *jsonErr
			if !errors.As(err, &e) {
				return jsonEncodeErr("ERROR", err.Error())
			}
			return err
		}
		return fmt.Errorf("Error: Section is %s %s", m.Section, err.Error())
	}
	return nil
}

func (m *Migration) do(do string) error {
	db, err := openDatabase(m.Driver, m.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := m.fetchApplied(db, do); err != nil {
		return err
	}

	if do == StatusDo {
		o, err := m.stringStatus()
		if len(o) > 0 {
			fmt.Print(o)
		}
		return err
	}

	applySources := m.status.fetchApplySources(do)
	if len(applySources) == 0 {
		a := fmt.Sprintf("Section %s has no version to migration.", m.Section)
		if m.JsonFormat {
			a = (jsonEncode(severityNotice, a))
		}
		fmt.Println(a)
		return nil
	}

	for _, v := range applySources {
		var mSQL, vSQL string
		switch do {
		case UpDo, ForceUpDo:
			mSQL = v.UpSQL
			vSQL = m.VersionSQLBuilder.InsretApply(v.Version)
		case DownDo:
			mSQL = v.DownSQL
			vSQL = m.VersionSQLBuilder.DeleteApply(v.Version)
		}
		if len(mSQL) == 0 {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			m.status.Error = err
			break
		}
		execSQL := fmt.Sprintf("%s%s", mSQL, vSQL)
		if _, err := tx.Exec(execSQL); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				panic(rerr)
			}
			m.status.Error = err
			break
		}
		if err := tx.Commit(); err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				panic(rerr)
			}
			m.status.Error = err
			break
		}

		v.Apply = true
	}

	o, err := m.stringApplied(do)
	if len(o) > 0 {
		fmt.Println(o)
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

func (s *status) fetchApplySources(do string) (ss []*Source) {
	switch do {
	case UpDo:
		ss = s.ApplySources
	case ForceUpDo:
		ss = append(s.UnappliedSources, s.ApplySources...)
	case DownDo:
		if s.CurrentSource != nil {
			return []*Source{s.CurrentSource}
		}
	}
	if ss == nil {
		return []*Source{}
	}
	return ss
}
