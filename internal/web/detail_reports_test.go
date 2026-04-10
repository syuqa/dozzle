package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/cardreports"
)

func TestParseJSONReportWithReportLinkColumn(t *testing.T) {
	payload := `{
		"title":"Modules",
		"type":"table",
		"columns":[
			{"key":"module","label":"Module"},
			{"key":"jira","label":"Jira","kind":"report_link","reportId":"jira-by-module","paramKey":"module","buttonLabel":"Open","passRow":true}
		],
		"rows":[{"module":"core","jira":"open"}]
	}`

	report, err := parseJSONReport(payload)
	if err != nil {
		t.Fatalf("parseJSONReport() error = %v", err)
	}

	if got, want := len(report.Columns), 2; got != want {
		t.Fatalf("columns len = %d, want %d", got, want)
	}
	if report.Columns[1].Kind != "report_link" {
		t.Fatalf("column kind = %q, want report_link", report.Columns[1].Kind)
	}
	if report.Columns[1].ReportID != "jira-by-module" {
		t.Fatalf("column reportId = %q", report.Columns[1].ReportID)
	}
	if got := report.Rows[0].Values["module"]; got != "core" {
		t.Fatalf("row module = %q", got)
	}
}

func TestExecuteJiraIssuesReport(t *testing.T) {
	jiraHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer secret-token" {
			t.Fatalf("unexpected auth header %q", auth)
		}
		var request jiraSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.JQL != `text ~ "\"module-core\""` {
			t.Fatalf("unexpected JQL %q", request.JQL)
		}

		payload, _ := json.Marshal(jiraSearchResponse{
			Issues: []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string `json:"summary"`
					Updated string `json:"updated"`
					Status  struct {
						Name string `json:"name"`
					} `json:"status"`
					Priority struct {
						Name string `json:"name"`
					} `json:"priority"`
					Assignee *struct {
						DisplayName string `json:"displayName"`
					} `json:"assignee"`
				} `json:"fields"`
			}{
				{
					Key: "CB-42",
					Fields: struct {
						Summary string `json:"summary"`
						Updated string `json:"updated"`
						Status  struct {
							Name string `json:"name"`
						} `json:"status"`
						Priority struct {
							Name string `json:"name"`
						} `json:"priority"`
						Assignee *struct {
							DisplayName string `json:"displayName"`
						} `json:"assignee"`
					}{
						Summary: "Core module issue",
						Updated: "2026-04-09T13:12:11.000+0300",
						Status: struct {
							Name string `json:"name"`
						}{Name: "In Progress"},
						Priority: struct {
							Name string `json:"name"`
						}{
							Name: "High",
						},
						Assignee: &struct {
							DisplayName string `json:"displayName"`
						}{DisplayName: "John Doe"},
					},
				},
			},
		})

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(payload))),
			Header:     make(http.Header),
		}, nil
	})}
	defer func() { jiraHTTPClient = http.DefaultClient }()

	t.Setenv("DOZZLE_JIRA_URL", "https://jira.example.com")
	t.Setenv("DOZZLE_JIRA_TOKEN", "secret-token")

	response, err := executeJiraIssuesReport(context.Background(), cardreports.Definition{
		ID:      "jira-by-module",
		Name:    "Jira by module",
		Command: `text ~ "\"{value}\""`,
	}, detailReportParams{Key: "module", Value: "module-core"})
	if err != nil {
		t.Fatalf("executeJiraIssuesReport() error = %v", err)
	}

	if got, want := response.Columns[0].Kind, "external_link"; got != want {
		t.Fatalf("first column kind = %q, want %q", got, want)
	}
	if got := response.Rows[0].Values["key"]; got != "CB-42" {
		t.Fatalf("key = %q", got)
	}
	if got := response.Rows[0].Values["url"]; got != "https://jira.example.com/browse/CB-42" {
		t.Fatalf("url = %q", got)
	}
	if got := response.Rows[0].Values["updated"]; got != "2026-04-09 13:12:11" {
		t.Fatalf("updated = %q", got)
	}
}

func TestExecuteJiraIssuesReportBasicAuth(t *testing.T) {
	jiraHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:secret-token"))
		if auth := r.Header.Get("Authorization"); auth != want {
			t.Fatalf("unexpected auth header %q", auth)
		}
		payload, _ := json.Marshal(jiraSearchResponse{})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(payload))),
			Header:     make(http.Header),
		}, nil
	})}
	defer func() { jiraHTTPClient = http.DefaultClient }()

	t.Setenv("DOZZLE_JIRA_URL", "https://jira.example.com")
	t.Setenv("DOZZLE_JIRA_TOKEN", "secret-token")
	t.Setenv("DOZZLE_JIRA_EMAIL", "user@example.com")

	if _, err := executeJiraIssuesReport(context.Background(), cardreports.Definition{
		ID:      "jira-basic",
		Name:    "Jira basic",
		Command: `text ~ "\"{value}\""`,
	}, detailReportParams{Value: "core"}); err != nil {
		t.Fatalf("executeJiraIssuesReport() error = %v", err)
	}
}

func TestParsePropertiesTemplateReport(t *testing.T) {
	content := `
org.unidata.mdm.system.cache.group=${CACHE_GROUP:unidata}
org.unidata.mdm.system.cache.password=${CACHE_PASSWORD:password}
org.unidata.mdm.system.cache.port=${CACHE_PORT:5701}
org.unidata.mdm.system.cache.port.autoincrement=${CACHE_PORT_AUTOINCREMENT:false}
`

	rows := parsePropertiesTemplateReport(content, []string{
		"CACHE_GROUP=custom-group",
		"CACHE_PORT=5801",
	}, []string{"org.unidata.mdm.system.cache.password*"})

	if got, want := len(rows), 3; got != want {
		t.Fatalf("rows len = %d, want %d", got, want)
	}
	if rows[0].Values["parameter"] != "org.unidata.mdm.system.cache.group" || rows[0].Values["value"] != "custom-group" {
		t.Fatalf("unexpected first row: %#v", rows[0].Values)
	}
	if rows[0].Values["source"] != "env" {
		t.Fatalf("first row source = %q", rows[0].Values["source"])
	}
	if rows[1].Values["value"] != "5801" {
		t.Fatalf("port value = %q", rows[1].Values["value"])
	}
	if rows[1].Values["source"] != "env" {
		t.Fatalf("second row source = %q", rows[1].Values["source"])
	}
	if rows[2].Values["value"] != "false" {
		t.Fatalf("default value = %q", rows[2].Values["value"])
	}
	if rows[2].Values["source"] != "default" {
		t.Fatalf("third row source = %q", rows[2].Values["source"])
	}
}

func TestParsePropertiesTemplateReportSupportsContinuationsAndColonSeparator(t *testing.T) {
	content := strings.Join([]string{
		"app.modules=core,\\",
		" api,\\",
		" billing",
		"app.title: ${APP_TITLE:Default title}",
		`escaped.key=value with escaped colon\:ok`,
	}, "\n")

	rows := parsePropertiesTemplateReport(content, []string{"APP_TITLE=Custom title"}, nil)
	if got, want := len(rows), 3; got != want {
		t.Fatalf("rows len = %d, want %d", got, want)
	}

	if got, want := rows[0].Values["parameter"], "app.modules"; got != want {
		t.Fatalf("first parameter = %q, want %q", got, want)
	}
	if got, want := rows[0].Values["value"], "core,api,billing"; got != want {
		t.Fatalf("first value = %q, want %q", got, want)
	}

	if got, want := rows[1].Values["parameter"], "app.title"; got != want {
		t.Fatalf("second parameter = %q, want %q", got, want)
	}
	if got, want := rows[1].Values["value"], "Custom title"; got != want {
		t.Fatalf("second value = %q, want %q", got, want)
	}
	if got, want := rows[1].Values["source"], "env"; got != want {
		t.Fatalf("second source = %q, want %q", got, want)
	}

	if got, want := rows[2].Values["value"], "value with escaped colon:ok"; got != want {
		t.Fatalf("third value = %q, want %q", got, want)
	}
}

func TestParseFileListingRows(t *testing.T) {
	output := strings.Join([]string{
		"2026-04-03 19:06:08.000000000 +0300|/opt/app/a.jar",
		"2026-04-03 19:07:09.000000000 +0300|/opt/app/b.jar",
		"2026-04-03 19:07:09.000000000 +0300|/opt/app/b.jar",
		"2026-04-03 19:08:10.000000000 +0300|/opt/app/skip.tmp",
	}, "\n")

	rows := parseFileListingRows(output, []string{"*.tmp"}, true, false, false)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if got := rows[0].Values["updated_at"]; got != "2026-04-03 19:06:08" {
		t.Fatalf("unexpected timestamp: %q", got)
	}
	if got := rows[0].Values["file"]; got != "a.jar" {
		t.Fatalf("unexpected file: %q", got)
	}
	if got := rows[0].Values["linked_report"]; got != "open" {
		t.Fatalf("unexpected linked report marker: %q", got)
	}
}

func TestParseFileListingRowsAllowsEmptyTimestamp(t *testing.T) {
	output := "|/opt/app/a.jar\n|/opt/app/b.jar\n"

	rows := parseFileListingRows(output, nil, false, false, false)
	if got, want := len(rows), 2; got != want {
		t.Fatalf("rows len = %d, want %d", got, want)
	}
	if got, want := rows[0].Values["updated_at"], ""; got != want {
		t.Fatalf("updated_at = %q, want empty", got)
	}
	if got, want := rows[0].Values["file"], "a.jar"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
}

func TestParseFileListingRowsFromPlainFindOutput(t *testing.T) {
	output := "/opt/app/a.jar\n/opt/app/b.jar\n"

	rows := parseFileListingRows(output, nil, true, false, false)
	if got, want := len(rows), 2; got != want {
		t.Fatalf("rows len = %d, want %d", got, want)
	}
	if got, want := rows[0].Values["updated_at"], ""; got != want {
		t.Fatalf("updated_at = %q, want empty", got)
	}
	if got, want := rows[0].Values["file"], "a.jar"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if got, want := rows[0].Values["linked_report"], "open"; got != want {
		t.Fatalf("linked_report = %q, want %q", got, want)
	}
}

func TestParseFileListingRowsWithSizeAndChecksum(t *testing.T) {
	output := "2026-04-03 19:06:08.000000000 +0300|2048|abc123|/opt/app/a.jar\n"

	rows := parseFileListingRows(output, nil, false, true, true)
	if got, want := len(rows), 1; got != want {
		t.Fatalf("rows len = %d, want %d", got, want)
	}
	if got, want := rows[0].Values["size"], "2.0 KB"; got != want {
		t.Fatalf("size = %q, want %q", got, want)
	}
	if got, want := rows[0].Values["checksum"], "abc123"; got != want {
		t.Fatalf("checksum = %q, want %q", got, want)
	}
}

func TestIsSensitiveEnvName(t *testing.T) {
	cases := map[string]bool{
		"CACHE_GROUP":        false,
		"JAVA_OPTS":          false,
		"CACHE_PASSWORD":     true,
		"client_secret":      true,
		"SERVICE_TOKEN":      true,
		"api_key":            true,
		"SPRING_AUTH_HEADER": true,
	}

	for input, expected := range cases {
		if got := isSensitiveEnvName(input); got != expected {
			t.Fatalf("isSensitiveEnvName(%q) = %v, want %v", input, got, expected)
		}
	}
}

func TestIsSensitiveEnvNameWithCustomPatterns(t *testing.T) {
	t.Setenv(customEnvMaskPatternsVar, "JWT,*_COOKIE,sessionid")

	cases := map[string]bool{
		"ACCESS_JWT":    true,
		"AUTH_COOKIE":   true,
		"SESSIONID":     true,
		"CACHE_GROUP":   false,
		"PUBLIC_COOKIE": true,
	}

	for input, expected := range cases {
		if got := isSensitiveEnvName(input); got != expected {
			t.Fatalf("isSensitiveEnvName(%q) = %v, want %v", input, got, expected)
		}
	}
}

func TestDetectFileContentFormat(t *testing.T) {
	cases := []struct {
		path     string
		content  string
		expected string
	}{
		{path: "/tmp/test.json", content: `{"ok":true}`, expected: "json"},
		{path: "/tmp/test.yaml", content: "name: app\n", expected: "yaml"},
		{path: "/tmp/test.xml", content: "<?xml version=\"1.0\"?><root/>", expected: "xml"},
		{path: "/tmp/test.html", content: "<html><body>ok</body></html>", expected: "html"},
		{path: "/tmp/test.conf", content: "PORT=8080\n", expected: "text"},
	}

	for _, tc := range cases {
		if got := detectFileContentFormat(tc.path, []byte(tc.content)); got != tc.expected {
			t.Fatalf("detectFileContentFormat(%q) = %q, want %q", tc.path, got, tc.expected)
		}
	}
}

func TestIsPreviewableTextFile(t *testing.T) {
	cases := map[string]bool{
		"/tmp/app.json": true,
		"/tmp/app.yaml": true,
		"/tmp/app.xml":  true,
		"/tmp/app.html": true,
		"/tmp/app.log":  false,
		"/tmp/app.jar":  false,
		"/tmp/app.pdf":  false,
	}

	for input, expected := range cases {
		if got := isPreviewableTextFile(input); got != expected {
			t.Fatalf("isPreviewableTextFile(%q) = %v, want %v", input, got, expected)
		}
	}
}

func TestIsPreviewableTextFileWithCustomFormats(t *testing.T) {
	t.Setenv(filePreviewFormatsEnvVar, "json,sql")

	cases := map[string]bool{
		"/tmp/app.json": true,
		"/tmp/app.sql":  true,
		"/tmp/app.yaml": false,
		"/tmp/app.log":  false,
	}

	for input, expected := range cases {
		if got := isPreviewableTextFile(input); got != expected {
			t.Fatalf("isPreviewableTextFile(%q) = %v, want %v", input, got, expected)
		}
	}
}

func TestFilePreviewMaxBytes(t *testing.T) {
	t.Setenv(filePreviewMaxBytesEnvVar, "4096")
	if got := filePreviewMaxBytes(); got != 4096 {
		t.Fatalf("filePreviewMaxBytes() = %d, want 4096", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
