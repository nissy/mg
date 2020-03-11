package mg

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func toJson(severity string, message interface{}) string {
	b, err := json.Marshal(
		map[string]interface{}{
			"time":     time.Now().Format(time.RFC3339Nano),
			"severity": severity,
			"message":  message,
		},
	)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (s *Status) sources(do int) []*Source {
	switch do {
	case UpDo:
		return s.AfterUnapplieds
	case DownDo:
		if s.CurrentVersion == 0 {
			return []*Source{}
		}
		return []*Source{s.CurrentApplied}
	}
	return nil
}

func (s *Status) displayApplys(do int) (string, error) {
	if s.JsonLog {
		m := map[string]interface{}{
			"section": s.Section,
			"current": s.CurrentVersion,
		}
		var doSources []*Source
		for _, v := range s.sources(do) {
			doSources = append(doSources, v)
			if !v.Apply {
				break
			}
		}
		m[doLabel(do)] = doSources
		if s.Error != nil {
			m["error"] = s.Error.Error()
			return "", errors.New(toJson("ERROR", m))
		}
		return toJson("INFO", m), nil
	}

	var out []string
	for _, v := range s.sources(do) {
		out = append(out, fmt.Sprintf("%s %d to %s is %s", state(v.Apply), v.Version, s.Section, v.File))
		if !v.Apply {
			break
		}
	}
	return strings.Join(out, "\n"), s.Error
}

func (s *Status) display() (string, error) {
	var err error
	if len(s.BeforeUnapplieds) > 0 {
		err = errors.New("Unapplied version exists before current version.")
	}
	if s.JsonLog {
		m := map[string]interface{}{
			"section": s.Section,
			"current": s.CurrentVersion,
		}
		if len(s.BeforeUnapplieds) > 0 {
			m["before_unapplieds"] = s.BeforeUnapplieds
		}
		if len(s.AfterUnapplieds) > 0 {
			m["after_unapplieds"] = s.AfterUnapplieds
		}
		if err != nil {
			m["error"] = err.Error()
			return "", errors.New(toJson("ERROR", m))
		}
		return toJson("INFO", m), nil
	}

	out := fmt.Sprintf("    current:\n        %d\n", s.CurrentVersion)
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
	return fmt.Sprintf("Version of %s:\n%s", s.Section, out), err
}

func state(apply bool) string {
	if apply {
		return "OK"
	}
	return "NG"
}

func doLabel(do int) string {
	switch do {
	case UpDo:
		return "up"
	case DownDo:
		return "down"
	case StatusDo:
		return "status"
	}

	return ""
}
