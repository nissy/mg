package mg

import (
	"bufio"
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

	defaultUpAnnotation   = "@migrate.up"
	defaultDownAnnotation = "@migrate.down"
	defaultVersionTable   = "migration_versions"
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
		UpAnnotation      string            `toml:"up_annotation"`
		DownAnnotation    string            `toml:"down_annotation"`
		Apply             bool              `toml:"-"`
		OutputFormat      string            `toml:"output_format"`
	}

	Source struct {
		UpSQL   string
		DownSQL string
		File    string
		Version uint64
		Apply   bool
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

	if len(m.VersionTable) == 0 {
		m.VersionTable = defaultVersionTable
	}
	if len(m.UpAnnotation) == 0 {
		m.UpAnnotation = defaultUpAnnotation
	}
	if len(m.DownAnnotation) == 0 {
		m.DownAnnotation = defaultDownAnnotation
	}
	if m.VersionSQLBuilder = FetchVersionSQLBuilder(m.Driver, m.VersionTable); m.VersionSQLBuilder == nil {
		return fmt.Errorf("Driver is %s does not exist.", m.Driver)
	}

	return nil
}

func (m *Migration) output(s *Source) string {
	switch strings.ToUpper(m.OutputFormat) {
	case "JSON":
		return fmt.Sprintf(
			`{"apply":%t,"version":%d,"section":"%s","file":"%s"}`,
			s.Apply, s.Version, m.Section, s.File,
		)
	}
	return fmt.Sprintf("%s %d to %s is %s", state(s.Apply), s.Version, m.Section, s.File)
}

func state(apply bool) string {
	if apply {
		return "OK"
	}
	return "NG"
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

	if err := db.Ping(); err != nil {
		return err
	}

	var lastVersion uint64
	if err := db.QueryRow(m.VersionSQLBuilder.FetchLastApplied()).Scan(&lastVersion); err != nil {
		switch do {
		case UpDo, DownDo:
			if _, err := db.Exec(m.VersionSQLBuilder.CreateTable()); err != nil {
				return err
			}
		case StatusDo:
			fmt.Printf("\x1b[31m%s\x1b[0m\n", err.Error())
		}
	}

	var unApplied []string
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
		case StatusDo:
			if lastVersion >= v.Version {
				continue
			}
			unApplied = append(unApplied,
				fmt.Sprintf("        \x1b[33m%d %s\x1b[0m\n", v.Version, v.File),
			)
			continue
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
			fmt.Println(m.output(v))
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
		fmt.Println(m.output(v))
	}

	switch do {
	case StatusDo:
		fmt.Printf("Version of %s:\n    current:\n        %d\n", m.Section, lastVersion)
		if len(unApplied) > 0 {
			fmt.Println("    \x1b[33munapplied:\x1b[0m")
			fmt.Print(strings.Join(unApplied, ""))
		}
	case UpDo, DownDo:
		if !m.Apply {
			fmt.Printf("%s has no version to migration.\n", m.Section)
		}
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
