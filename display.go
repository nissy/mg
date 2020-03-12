package mg

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	severityDefault   = "DEFAULT"
	severityDebug     = "DEBUG"
	severityInfo      = "INFO"
	severityNotice    = "NOTICE"
	severityWarning   = "WARNING"
	severityError     = "ERROR"
	severityCritical  = "CRITICAL"
	severityAlert     = "ALERT"
	severityEmergency = "EMERGENCY"
)

type jsonErr struct {
	output string
}

func (e *jsonErr) Error() string {
	return e.output
}

func toJsonErr(severity string, message interface{}) error {
	return &jsonErr{
		output: toJson(severity, message),
	}
}

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

func (s *status) unapplieds(do string) []*Source {
	switch do {
	case UpDo:
		return s.AfterUnapplieds
	case DownDo:
		if s.CurrentApplied == nil {
			return []*Source{}
		}
		return []*Source{s.CurrentApplied}
	}
	return nil
}

func (m *Migration) displayApply(do string) (string, error) {
	if m.JsonFormat {
		j := map[string]interface{}{
			"section": m.Section,
			"current": m.status.CurrentVersion,
		}
		var doSources []*Source
		for _, v := range m.status.unapplieds(do) {
			doSources = append(doSources, v)
			if !v.Apply {
				break
			}
		}
		j[do] = doSources
		if m.status.Error != nil {
			j["error"] = m.status.Error.Error()
			return "", toJsonErr(severityCritical, j)
		}
		return toJson(severityNotice, j), nil
	}

	var out []string
	for _, v := range m.status.unapplieds(do) {
		out = append(out, fmt.Sprintf("%s %d to %s is %s", state(v.Apply), v.Version, m.Section, v.File))
		if !v.Apply {
			break
		}
	}
	return strings.Join(out, "\n"), m.status.Error
}

func (m *Migration) displayStatus() (string, error) {
	var err error
	if len(m.status.BeforeUnapplieds) > 0 {
		err = errors.New("Unapplied version exists before current version.")
	}
	if m.JsonFormat {
		j := map[string]interface{}{
			"section": m.Section,
			"current": m.status.CurrentVersion,
		}
		if len(m.status.BeforeUnapplieds) > 0 {
			j["before_unapplieds"] = m.status.BeforeUnapplieds
		}
		if len(m.status.AfterUnapplieds) > 0 {
			j["after_unapplieds"] = m.status.AfterUnapplieds
		}
		if err != nil {
			j["error"] = err.Error()
			return "", toJsonErr(severityError, j)
		}
		return toJson(severityInfo, j), nil
	}

	out := fmt.Sprintf("    current:\n        %d\n", m.status.CurrentVersion)
	if len(m.status.BeforeUnapplieds) > 0 {
		var befores []string
		for _, v := range m.status.BeforeUnapplieds {
			befores = append(befores, fmt.Sprintf("%d %s", v.Version, v.File))
		}
		out = fmt.Sprintf("    \x1b[31munapplied version before current:\n%s\x1b[0m%s", fmt.Sprintf("        %s\n", strings.Join(befores, "\n        ")), out)
	}
	if len(m.status.AfterUnapplieds) > 0 {
		var afters []string
		for _, v := range m.status.AfterUnapplieds {
			afters = append(afters, fmt.Sprintf("%d %s", v.Version, v.File))
		}
		out = fmt.Sprintf("%s    \x1b[33munapplied:\n%s\x1b[0m", out, fmt.Sprintf("        %s\n", strings.Join(afters, "\n        ")))
	}
	return fmt.Sprintf("Version of %s:\n%s", m.Section, out), err
}

func state(apply bool) string {
	if apply {
		return "OK"
	}
	return "NG"
}
