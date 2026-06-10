package renderers

import (
	"bufio"
	"encoding/json"
	"io"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Logs colorizes container log output by detecting JSON, logfmt or plain-text
// formats and highlighting the log level.
type Logs struct{}

func (Logs) Run(ctx Context) int {
	printHeader(ctx, accentLogs(), "")
	out("")

	consume := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			out(colorizeLogLine(sc.Text()))
		}
	}
	return intercept.RunStreamed(ctx.Real, ctx.Args, consume, consume)
}

var logfmtRe = regexp.MustCompile(`(\w+)=("[^"]*"|\S+)`)

func colorizeLogLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}
	// 1) JSON structured.
	if strings.HasPrefix(strings.TrimSpace(line), "{") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			level := firstString(m, "level", "lvl", "severity")
			msg := firstString(m, "msg", "message")
			ts := firstString(m, "ts", "time", "timestamp", "@timestamp")
			if level != "" || msg != "" {
				return renderLog(ts, level, msg, kvPairsFromMap(m))
			}
		}
	}
	// 2) logfmt (has key=value pairs including a level).
	if strings.Contains(line, "level=") || strings.Contains(line, "lvl=") {
		fields := map[string]string{}
		for _, m := range logfmtRe.FindAllStringSubmatch(line, -1) {
			fields[m[1]] = strings.Trim(m[2], `"`)
		}
		level := fields["level"]
		if level == "" {
			level = fields["lvl"]
		}
		msg := fields["msg"]
		if msg == "" {
			msg = fields["message"]
		}
		ts := fields["ts"]
		if ts == "" {
			ts = fields["time"]
		}
		var kv []string
		for k, v := range fields {
			if k == "level" || k == "lvl" || k == "msg" || k == "message" || k == "ts" || k == "time" {
				continue
			}
			kv = append(kv, style.Dim.Render(k+"=")+style.Muted.Render(v))
		}
		return renderLog(ts, level, msg, kv)
	}
	// 3) plain text — scan for a level keyword.
	if lvl := scanLevel(line); lvl != "" {
		return levelStyle(lvl).Render(line)
	}
	return style.Text.Render(line)
}

func renderLog(ts, level, msg string, kv []string) string {
	var b strings.Builder
	if ts != "" {
		b.WriteString(style.Dim.Render(ts) + "  ")
	}
	if level != "" {
		lvl := strings.ToUpper(level)
		if len(lvl) > 5 {
			lvl = lvl[:5]
		}
		b.WriteString(levelStyle(level).Render(style.PadRight(lvl, 5)) + "  ")
	}
	b.WriteString(style.Text.Render(msg))
	for _, p := range kv {
		b.WriteString("  " + p)
	}
	return b.String()
}

func levelStyle(level string) lipgloss.Style {
	switch strings.ToUpper(level) {
	case "ERROR", "ERR", "FATAL", "CRIT", "CRITICAL":
		return style.Red.Bold(true)
	case "WARN", "WARNING":
		return style.Yellow
	case "INFO":
		return style.Blue
	case "DEBUG", "TRACE":
		return style.Dim
	}
	return style.Text
}

func scanLevel(line string) string {
	upper := strings.ToUpper(line)
	for _, lvl := range []string{"FATAL", "ERROR", "CRIT", "WARN", "INFO", "DEBUG", "TRACE"} {
		if strings.Contains(upper, lvl) {
			return lvl
		}
	}
	return ""
}

func firstString(m map[string]any, keys ...string) string {
	for k, v := range m {
		for _, want := range keys {
			if strings.EqualFold(k, want) {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

func kvPairsFromMap(m map[string]any) []string {
	var kv []string
	for k, v := range m {
		lk := strings.ToLower(k)
		switch lk {
		case "level", "lvl", "severity", "msg", "message", "ts", "time", "timestamp", "@timestamp":
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		kv = append(kv, style.Dim.Render(k+"=")+style.Muted.Render(s))
	}
	return kv
}
