package main

import (
	"strings"
	"testing"
	"time"
)

func TestNotifyNextRun(t *testing.T) {
	for _, tc := range []struct{ now, want string }{
		{"2026-11-03T00:59:00Z", "2026-11-03T08:00:00+07:00"},
		{"2026-11-03T01:00:00Z", "2026-11-04T08:00:00+07:00"},
		{"2026-12-31T23:00:00+07:00", "2027-01-01T08:00:00+07:00"},
	} {
		now, _ := time.Parse(time.RFC3339, tc.now)
		if got := nextNotifyRun("08:00", now).Format(time.RFC3339); got != tc.want {
			t.Fatalf("got %s want %s", got, tc.want)
		}
	}
	s := defaultSchedule()
	s.Hour = "24:00"
	if s.validate() == nil {
		t.Fatal("accepted 24:00")
	}
	s.Hour = "08:30"
	if s.validate() == nil {
		t.Fatal("accepted non-hour slot")
	}
}
func TestNotifyCustomTemplate(t *testing.T) {
	data := notifyTemplateData{Name: "Marcell", Urgent: 3, High: 3, HighTasks: []Todo{{Title: "Deploy"}, {Title: "Review"}}}
	got, err := renderNotifyTemplate("Hello {{.Name}}, urgent tasks: {{.Urgent}}.{{range .HighTasks}}\n- {{.Title}}{{end}}", data)
	if err != nil || got != "Hello Marcell, urgent tasks: 3.\n- Deploy\n- Review" {
		t.Fatalf("%q %v", got, err)
	}
	for _, source := range []string{"{{.Unknown}}", "{{", `{{define "loop"}}{{template "loop" .}}{{end}}{{template "loop" .}}`, "{{range .Tasks}}{{range $.Tasks}}{{end}}{{end}}"} {
		if _, err := renderNotifyTemplate(source, data); err == nil {
			t.Fatalf("accepted invalid/unbounded template %q", source)
		}
	}
	if _, err := renderNotifyTemplate(strings.Repeat("x", 100001), data); err == nil {
		t.Fatal("unbounded output")
	}
}

func TestDefaultHTMLTable(t *testing.T) {
	s := defaultSchedule()
	data := notifyTemplateData{Name: "Marcell", Total: 100, High: 100, Urgent: 100}
	for i := 0; i < 100; i++ {
		data.Tasks = append(data.Tasks, Todo{ID: int64(i + 1), Title: strings.Repeat("<>&", 60), Priority: "high"})
	}
	rendered, err := renderNotifyTemplate(s.Body, data, "html")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered, "<td><>&") || !strings.Contains(rendered, "&lt;&gt;&amp;") || !strings.Contains(rendered, "Total: 100") {
		t.Fatal("table data not escaped or totals missing")
	}
	if err := validateEmailContent("Reminder", rendered, "html"); err != nil {
		t.Fatal(err)
	}
	empty, err := renderNotifyTemplate(s.Body, notifyTemplateData{}, "html")
	if err != nil || !strings.Contains(empty, `colspan="5" align="center"`) {
		t.Fatal("empty-state table missing", err)
	}
}
