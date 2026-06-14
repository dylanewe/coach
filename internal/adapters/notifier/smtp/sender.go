package smtp

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/dylanewe/coach/internal/domain"
)

// Sender implements ports.Notifier via SMTP.
type Sender struct {
	host string
	port int
	auth smtp.Auth
	from string
}

// NewSender creates an SMTP notifier.
func NewSender(host string, port int, user, pass, from string) *Sender {
	var auth smtp.Auth
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	return &Sender{
		host: host,
		port: port,
		auth: auth,
		from: from,
	}
}

// SendRunAnalysis emails a single-run analysis.
func (s *Sender) SendRunAnalysis(ctx context.Context, to string, analysis domain.RunAnalysis, activity domain.Activity) error {
	subject := fmt.Sprintf("Run Analysis: %s", activity.Name)
	body, err := renderRunAnalysisHTML(activity, analysis)
	if err != nil {
		return fmt.Errorf("render analysis: %w", err)
	}
	return s.send(to, subject, body)
}

// SendWeeklyReport emails the weekly summary.
func (s *Sender) SendWeeklyReport(ctx context.Context, to string, report domain.WeeklyReport) error {
	subject := fmt.Sprintf("Weekly Running Report: %s", report.WeekStart.Format("2006-01-02"))
	body, err := renderWeeklyReportHTML(report)
	if err != nil {
		return fmt.Errorf("render weekly report: %w", err)
	}
	return s.send(to, subject, body)
}

func (s *Sender) send(to, subject, body string) error {
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return smtp.SendMail(addr, s.auth, s.from, []string{to}, msg)
}

func renderRunAnalysisHTML(activity domain.Activity, analysis domain.RunAnalysis) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Run Analysis</title></head>
<body>
<h2>{{.Activity.Name}}</h2>
<p><strong>Date:</strong> {{.Activity.StartDateLocal.Format "2006-01-02"}}<br>
<strong>Distance:</strong> {{printf "%.2f" .Activity.DistanceMeters}} km<br>
<strong>Time:</strong> {{.Activity.MovingTime}}</p>
<h3>Summary</h3>
<p>{{.Analysis.Summary}}</p>
<h3>Positives</h3>
<ul>{{range .Analysis.Positives}}<li>{{.}}</li>{{end}}</ul>
<h3>Areas for Improvement</h3>
<ul>{{range .Analysis.AreasForImprovement}}<li>{{.}}</li>{{end}}</ul>
<h3>Suggested Next Session</h3>
<p>{{.Analysis.SuggestedNextSession}}</p>
<p><em>Fatigue Score: {{.Analysis.FatigueScore}}/10</em></p>
</body>
</html>`
	t := template.Must(template.New("run").Parse(tmpl))
	var buf bytes.Buffer
	data := struct {
		Activity activityView
		Analysis domain.RunAnalysis
	}{
		Activity: activityView{
			Name:           activity.Name,
			StartDateLocal: activity.StartDateLocal,
			DistanceMeters: activity.Distance / 1000,
			MovingTime:     fmt.Sprintf("%d:%02d", activity.MovingTime/60, activity.MovingTime%60),
		},
		Analysis: analysis,
	}
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderWeeklyReportHTML(report domain.WeeklyReport) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Weekly Report</title></head>
<body>
<h2>Weekly Report: {{.WeekStart.Format "2006-01-02"}} to {{.WeekEnd.Format "2006-01-02"}}</h2>
<h3>Summary</h3>
<p>{{.Summary}}</p>
<h3>Recommendations</h3>
<ul>{{range .Recommendations}}<li>{{.}}</li>{{end}}</ul>
<h3>Next Week's Plan</h3>
<table border="1" cellpadding="6">
<tr><th>Day</th><th>Name</th><th>Description</th><th>Target</th></tr>
{{range .NextWeekWorkouts}}
<tr><td>{{.Name}}</td><td>{{.Description}}</td><td>{{.Target}}</td></tr>
{{end}}
</table>
</body>
</html>`
	t := template.Must(template.New("weekly").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, report); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type activityView struct {
	Name           string
	StartDateLocal interface{}
	DistanceMeters float64
	MovingTime     string
}
