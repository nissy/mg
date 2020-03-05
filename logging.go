package mg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type (
	logging struct {
		JSON    bool
		Section string
	}
)

func (m *Migration) NewLogging() *logging {
	return &logging{
		JSON:    m.JsonLog,
		Section: m.Section,
	}
}

func (l *logging) toJson(i interface{}) string {
	b := bytes.NewBuffer([]byte{})
	if err := json.NewEncoder(b).Encode(i); err != nil {
		panic(err)
	}
	return b.String()
}

func (l *logging) source(s *Source) string {
	if l.JSON {
		return fmt.Sprintf(
			`{"apply":%t,"version":%d,"section":"%s","file":"%s"}`,
			s.Apply, s.Version, l.Section, s.File,
		)
	}

	return fmt.Sprintf("%s %d to %s is %s", state(s.Apply), s.Version, l.Section, s.File)
}

func (l *logging) status(s *Status) string {
	if l.JSON {
		return l.toJson(s)
	}

	out := fmt.Sprintf("    current:\n        %d\n", s.Current)
	if len(s.BeforeUnapplieds) > 0 {
		var befores []string
		for _, v := range s.BeforeUnapplieds {
			befores = append(befores, fmt.Sprintf("%d %s", v.Version, v.File))
		}
		out = fmt.Sprintf("    \x1b[31munapplied version before current:\n%s\x1b[0m%s", fmt.Sprintf("        %s\n", strings.Join(befores, "\n        ")), out)
	}
	if len(s.AfterUnapplieds) > 0 {
		var afters []string
		for _, v := range s.AfterUnapplieds {
			afters = append(afters, fmt.Sprintf("%d %s", v.Version, v.File))
		}
		out = fmt.Sprintf("%s    \x1b[33munapplied:\n%s\x1b[0m", out, fmt.Sprintf("        %s\n", strings.Join(afters, "\n        ")))
	}
	return fmt.Sprintf("Version of %s:\n%s", l.Section, out)
}

func state(apply bool) string {
	if apply {
		return "OK"
	}
	return "NG"
}
