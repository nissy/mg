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

func jsonEncodeErr(severity string, message interface{}) error {
	return &jsonErr{
		output: jsonEncode(severity, message),
	}
}

func jsonEncode(severity string, message interface{}) string {
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

func (m *Migration) stringApplied(do string) (string, error) {
	if m.JsonFormat {
		j := map[string]interface{}{
			"section": m.Section,
			"current": m.status.CurrentVersion,
		}
		var ss []*Source
		for _, v := range m.status.fetchApplySources(do) {
			ss = append(ss, v)
			if !v.Apply {
				break
			}
		}
		j[do] = ss
		if m.status.Error != nil {
			j["error"] = m.status.Error.Error()
			return "", jsonEncodeErr(severityCritical, j)
		}
		return jsonEncode(severityNotice, j), nil
	}

	var out []string
	for _, v := range m.status.fetchApplySources(do) {
		out = append(out, fmt.Sprintf("%s %d to %s is %s", state(v.Apply), v.Version, m.Section, v.File))
		if !v.Apply {
			break
		}
	}
	return strings.Join(out, "\n"), m.status.Error
}

func (m *Migration) stringStatus() (string, error) {
	var err error
	if len(m.status.UnappliedSources) > 0 {
		err = errors.New("Unapplied version exists before current version.")
	}
	if m.JsonFormat {
		j := map[string]interface{}{
			"section": m.Section,
			"current": m.status.CurrentVersion,
		}
		if len(m.status.UnappliedSources) > 0 {
			j["unapplied"] = m.status.UnappliedSources
		}
		if len(m.status.ApplySources) > 0 {
			j["apply"] = m.status.ApplySources
		}
		if err != nil {
			j["error"] = err.Error()
			return "", jsonEncodeErr(severityError, j)
		}
		return jsonEncode(severityInfo, j), nil
	}

	out := fmt.Sprintf("    current:\n        %d\n", m.status.CurrentVersion)
	if len(m.status.UnappliedSources) > 0 {
		var befores []string
		for _, v := range m.status.UnappliedSources {
			befores = append(befores, fmt.Sprintf("%d %s", v.Version, v.File))
		}
		out = fmt.Sprintf("    \x1b[31munapplied:\n%s\x1b[0m%s", fmt.Sprintf("        %s\n", strings.Join(befores, "\n        ")), out)
	}
	if len(m.status.ApplySources) > 0 {
		var afters []string
		for _, v := range m.status.ApplySources {
			afters = append(afters, fmt.Sprintf("%d %s", v.Version, v.File))
		}
		out = fmt.Sprintf("%s    \x1b[33mapply:\n%s\x1b[0m", out, fmt.Sprintf("        %s\n", strings.Join(afters, "\n        ")))
	}
	return fmt.Sprintf("Version of %s:\n%s", m.Section, out), err
}

func state(apply bool) string {
	if apply {
		return "OK"
	}
	return "NG"
}
