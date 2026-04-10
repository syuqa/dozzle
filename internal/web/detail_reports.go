package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/amir20/dozzle/internal/cardreports"
	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type detailReportColumn struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Kind        string `json:"kind,omitempty"`
	ReportID    string `json:"reportId,omitempty"`
	ParamKey    string `json:"paramKey,omitempty"`
	ButtonLabel string `json:"buttonLabel,omitempty"`
	PassRow     bool   `json:"passRow,omitempty"`
	URLKey      string `json:"urlKey,omitempty"`
}

type detailReportRow struct {
	Values map[string]string `json:"values"`
}

type detailReportResponse struct {
	Title       string               `json:"title"`
	Type        string               `json:"type"`
	Format      string               `json:"format,omitempty"`
	Columns     []detailReportColumn `json:"columns,omitempty"`
	Rows        []detailReportRow    `json:"rows,omitempty"`
	Text        string               `json:"text,omitempty"`
	GeneratedAt time.Time            `json:"generatedAt"`
	DurationMs  int64                `json:"durationMs"`
	Report      string               `json:"report"`
}

type cachedDetailReport struct {
	ExpiresAt time.Time
	Response  detailReportResponse
}

type detailReportParams struct {
	Key     string
	Value   string
	RowJSON string
}

const containerReportIdentityLabel = "dev.dozzle.link-id"
const (
	defaultMaxFileContentBytes = 128 * 1024
	filePreviewFormatsEnvVar   = "DOZZLE_FILE_PREVIEW_FORMATS"
	filePreviewMaxBytesEnvVar  = "DOZZLE_FILE_PREVIEW_MAX_BYTES"
)

var jiraHTTPClient = http.DefaultClient

func (h *handler) runContainerDetailReport(w http.ResponseWriter, r *http.Request) {
	if h.cardReportManager == nil {
		writeError(w, http.StatusServiceUnavailable, "card report manager is not configured")
		return
	}

	host := hostKey(r)
	containerID := chi.URLParam(r, "id")
	reportID := chi.URLParam(r, "reportId")
	report, ok := h.cardReportManager.Get(reportID)
	if !ok || !report.Enabled {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	log.Debug().Str("host", host).Str("container", containerID).Str("report", reportID).Msg("running container detail report")

	containerService, err := h.hostService.FindContainer(host, containerID, h.resolveLabels(r))
	if err != nil {
		log.Warn().Err(err).Str("host", host).Str("container", containerID).Str("report", reportID).Msg("failed to resolve container for detail report")
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	params := detailReportParams{
		Key:     strings.TrimSpace(r.URL.Query().Get("field")),
		Value:   strings.TrimSpace(r.URL.Query().Get("key")),
		RowJSON: strings.TrimSpace(r.URL.Query().Get("row")),
	}

	cacheKey := detailReportCacheKey(report, hostKey(r), containerService.Container, params)
	forceRefresh := r.URL.Query().Get("force") == "1"
	if forceRefresh {
		h.reportCache.Delete(cacheKey)
	} else if report.RefreshMode != cardreports.RefreshModeOnOpen {
		if cached, ok := h.reportCache.Load(cacheKey); ok {
			entry := cached.(cachedDetailReport)
			if entry.ExpiresAt.IsZero() || time.Now().Before(entry.ExpiresAt) {
				writeJSON(w, http.StatusOK, entry.Response)
				return
			}
			h.reportCache.Delete(cacheKey)
		}
		if report.RefreshMode == cardreports.RefreshModeManual {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	result, err := h.executeContainerReport(r.Context(), containerService, report, params)
	if err != nil {
		log.Warn().Err(err).Str("host", host).Str("container", containerID).Str("report", reportID).Msg("container detail report failed")
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if report.RefreshMode == cardreports.RefreshModeContainerUpdate || report.RefreshMode == cardreports.RefreshModeManual || report.CacheTTLSeconds > 0 {
		expiresAt := time.Time{}
		if report.RefreshMode == cardreports.RefreshModeTTL {
			expiresAt = time.Now().Add(time.Duration(report.CacheTTLSeconds) * time.Second)
		}
		h.reportCache.Store(cacheKey, cachedDetailReport{
			ExpiresAt: expiresAt,
			Response:  result,
		})
	}

	log.Debug().Str("host", host).Str("container", containerID).Str("report", reportID).Str("type", result.Type).Msg("container detail report completed")
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) executeContainerReport(parent context.Context, containerService *container_support.ContainerService, report cardreports.Definition, params detailReportParams) (detailReportResponse, error) {
	timeout := time.Duration(report.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	startedAt := time.Now()
	response := detailReportResponse{
		Title:       report.Name,
		GeneratedAt: time.Now(),
		DurationMs:  time.Since(startedAt).Milliseconds(),
		Report:      report.ID,
	}

	switch report.Parser {
	case cardreports.ParserJSON:
		stdout, err := executeCommandReport(ctx, containerService, report.Command, params)
		if err != nil {
			return detailReportResponse{}, err
		}
		parsed, err := parseJSONReport(stdout.String())
		if err != nil {
			return detailReportResponse{}, err
		}
		response.Type = parsed.Type
		response.Title = firstNonEmpty(strings.TrimSpace(parsed.Title), response.Title)
		response.Columns = parsed.Columns
		response.Rows = parsed.Rows
		response.Text = strings.TrimSpace(parsed.Text)
	case cardreports.ParserJarModules:
		stdout, err := executeCommandReport(ctx, containerService, report.Command, params)
		if err != nil {
			return detailReportResponse{}, err
		}
		response.Type = "table"
		response.Columns = []detailReportColumn{
			{Key: "module", Label: "module"},
			{Key: "version", Label: "version"},
		}
		response.Rows = parseJarModulesReport(stdout.String())
	case cardreports.ParserJiraIssues:
		jiraResponse, err := executeJiraIssuesReport(ctx, report, params)
		if err != nil {
			return detailReportResponse{}, err
		}
		response.Type = "table"
		response.Columns = jiraResponse.Columns
		response.Rows = jiraResponse.Rows
		response.Title = firstNonEmpty(jiraResponse.Title, response.Title)
	case cardreports.ParserPropertiesTemplate:
		propertiesResponse, err := executePropertiesTemplateReport(ctx, containerService, report)
		if err != nil {
			return detailReportResponse{}, err
		}
		response.Type = "table"
		response.Columns = propertiesResponse.Columns
		response.Rows = propertiesResponse.Rows
		response.Title = firstNonEmpty(propertiesResponse.Title, response.Title)
	case cardreports.ParserFileListing:
		fileListingResponse, err := executeFileListingReport(ctx, containerService, report)
		if err != nil {
			return detailReportResponse{}, err
		}
		response.Type = "table"
		response.Columns = fileListingResponse.Columns
		response.Rows = fileListingResponse.Rows
		response.Title = firstNonEmpty(fileListingResponse.Title, response.Title)
	case cardreports.ParserFileContent:
		fileContentResponse, err := executeFileContentReport(ctx, containerService, report)
		if err != nil {
			return detailReportResponse{}, err
		}
		response.Type = "text"
		response.Text = fileContentResponse.Text
		response.Format = fileContentResponse.Format
		response.Title = firstNonEmpty(fileContentResponse.Title, response.Title)
	default:
		stdout, err := executeCommandReport(ctx, containerService, report.Command, params)
		if err != nil {
			return detailReportResponse{}, err
		}
		response.Type = "text"
		response.Text = strings.TrimSpace(stdout.String())
	}

	return response, nil
}

func executeCommandReport(ctx context.Context, containerService *container_support.ContainerService, command string, params detailReportParams) (*bytes.Buffer, error) {
	var stdout bytes.Buffer
	execArgs := []string{"sh", "-lc", command}
	if params.Key != "" || params.Value != "" || params.RowJSON != "" {
		envArgs := []string{
			"DOZZLE_REPORT_PARAM_KEY=" + params.Key,
			"DOZZLE_REPORT_PARAM_VALUE=" + params.Value,
			"DOZZLE_REPORT_ROW_JSON=" + params.RowJSON,
		}
		execArgs = append([]string{"env"}, append(envArgs, execArgs...)...)
	}
	if err := containerService.Exec(ctx, execArgs, &closedExecEventReader{}, &stdout); err != nil {
		return nil, err
	}
	return &stdout, nil
}

var jarVersionPattern = regexp.MustCompile(`^(?P<module>.+)-(?P<version>\d[\w.\-+]*)\.jar$`)

func parseJarModulesReport(output string) []detailReportRow {
	rows := make([]detailReportRow, 0)
	seen := make(map[string]struct{})
	for _, line := range strings.Split(output, "\n") {
		name := filepath.Base(strings.TrimSpace(line))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		matches := jarVersionPattern.FindStringSubmatch(name)
		if len(matches) == 0 {
			rows = append(rows, detailReportRow{Values: map[string]string{"module": name, "version": ""}})
			continue
		}

		rows = append(rows, detailReportRow{Values: map[string]string{"module": matches[1], "version": matches[2]}})
	}
	return rows
}

type jsonReportPayload struct {
	Title   string `json:"title"`
	Type    string `json:"type"`
	Text    string `json:"text"`
	Columns []detailReportColumn
	Rows    []detailReportRow
}

func parseJSONReport(output string) (jsonReportPayload, error) {
	type rawPayload struct {
		Title   string          `json:"title"`
		Type    string          `json:"type"`
		Text    string          `json:"text"`
		Columns json.RawMessage `json:"columns"`
		Rows    json.RawMessage `json:"rows"`
	}
	type columnPayload struct {
		Key         string `json:"key"`
		Label       string `json:"label"`
		Kind        string `json:"kind"`
		ReportID    string `json:"reportId"`
		ParamKey    string `json:"paramKey"`
		ButtonLabel string `json:"buttonLabel"`
		PassRow     bool   `json:"passRow"`
		URLKey      string `json:"urlKey"`
	}

	raw := rawPayload{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &raw); err != nil {
		return jsonReportPayload{}, fmt.Errorf("invalid json report output: %w", err)
	}

	payload := jsonReportPayload{
		Title: strings.TrimSpace(raw.Title),
		Type:  strings.TrimSpace(raw.Type),
		Text:  strings.TrimSpace(raw.Text),
	}

	switch payload.Type {
	case "text":
	case "table":
		if len(raw.Columns) == 0 {
			return jsonReportPayload{}, fmt.Errorf("json report table is missing columns")
		}
		var stringColumns []string
		if err := json.Unmarshal(raw.Columns, &stringColumns); err == nil {
			payload.Columns = make([]detailReportColumn, 0, len(stringColumns))
			for _, column := range stringColumns {
				label := strings.TrimSpace(column)
				if label == "" {
					continue
				}
				payload.Columns = append(payload.Columns, detailReportColumn{Key: label, Label: label})
			}
		} else {
			var objectColumns []columnPayload
			if err := json.Unmarshal(raw.Columns, &objectColumns); err != nil {
				return jsonReportPayload{}, fmt.Errorf("json report table has invalid columns")
			}
			payload.Columns = make([]detailReportColumn, 0, len(objectColumns))
			for index, column := range objectColumns {
				key := strings.TrimSpace(column.Key)
				label := firstNonEmpty(strings.TrimSpace(column.Label), key)
				if key == "" {
					key = fmt.Sprintf("column_%d", index)
				}
				if label == "" {
					label = key
				}
				payload.Columns = append(payload.Columns, detailReportColumn{
					Key:         key,
					Label:       label,
					Kind:        strings.TrimSpace(column.Kind),
					ReportID:    strings.TrimSpace(column.ReportID),
					ParamKey:    strings.TrimSpace(column.ParamKey),
					ButtonLabel: strings.TrimSpace(column.ButtonLabel),
					PassRow:     column.PassRow,
					URLKey:      strings.TrimSpace(column.URLKey),
				})
			}
		}
		if len(payload.Columns) == 0 {
			return jsonReportPayload{}, fmt.Errorf("json report table is missing columns")
		}
		var objectRows []map[string]any
		if err := json.Unmarshal(raw.Rows, &objectRows); err == nil {
			payload.Rows = make([]detailReportRow, 0, len(objectRows))
			for _, row := range objectRows {
				values := make(map[string]string, len(payload.Columns))
				for _, column := range payload.Columns {
					values[column.Key] = stringifyReportValue(row[column.Key])
				}
				payload.Rows = append(payload.Rows, detailReportRow{Values: values})
			}
			break
		}
		var arrayRows [][]any
		if err := json.Unmarshal(raw.Rows, &arrayRows); err == nil {
			payload.Rows = make([]detailReportRow, 0, len(arrayRows))
			for _, row := range arrayRows {
				values := make(map[string]string, len(payload.Columns))
				for index, column := range payload.Columns {
					if index < len(row) {
						values[column.Key] = stringifyReportValue(row[index])
					} else {
						values[column.Key] = ""
					}
				}
				payload.Rows = append(payload.Rows, detailReportRow{Values: values})
			}
			break
		}
		return jsonReportPayload{}, fmt.Errorf("json report table has invalid rows")
	default:
		return jsonReportPayload{}, fmt.Errorf("json report has unsupported type %q", payload.Type)
	}

	return payload, nil
}

type jiraSearchRequest struct {
	JQL        string   `json:"jql"`
	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

type jiraSearchResponse struct {
	Issues []struct {
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
	} `json:"issues"`
}

func executeJiraIssuesReport(ctx context.Context, report cardreports.Definition, params detailReportParams) (detailReportResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("DOZZLE_JIRA_URL")), "/")
	if baseURL == "" {
		return detailReportResponse{}, fmt.Errorf("DOZZLE_JIRA_URL is not configured")
	}
	token := strings.TrimSpace(os.Getenv("DOZZLE_JIRA_TOKEN"))
	if token == "" {
		return detailReportResponse{}, fmt.Errorf("DOZZLE_JIRA_TOKEN is not configured")
	}

	jql := expandJiraTemplate(report.Command, params)
	if strings.TrimSpace(jql) == "" {
		return detailReportResponse{}, fmt.Errorf("jira report query is empty")
	}

	maxResults := 20
	if raw := strings.TrimSpace(os.Getenv("DOZZLE_JIRA_MAX_RESULTS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			maxResults = parsed
		}
	}

	body, err := json.Marshal(jiraSearchRequest{
		JQL:        jql,
		MaxResults: maxResults,
		Fields:     []string{"summary", "status", "assignee", "updated", "priority"},
	})
	if err != nil {
		return detailReportResponse{}, fmt.Errorf("marshal jira request: %w", err)
	}

	endpoint := baseURL + "/rest/api/3/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return detailReportResponse{}, fmt.Errorf("create jira request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if email := strings.TrimSpace(os.Getenv("DOZZLE_JIRA_EMAIL")); email != "" {
		req.SetBasicAuth(email, token)
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := jiraHTTPClient.Do(req)
	if err != nil {
		return detailReportResponse{}, fmt.Errorf("jira request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return detailReportResponse{}, fmt.Errorf("jira request failed: %s", strings.TrimSpace(string(payload)))
	}

	var decoded jiraSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		return detailReportResponse{}, fmt.Errorf("decode jira response: %w", err)
	}

	rows := make([]detailReportRow, 0, len(decoded.Issues))
	for _, issue := range decoded.Issues {
		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		rows = append(rows, detailReportRow{
			Values: map[string]string{
				"key":      issue.Key,
				"url":      baseURL + "/browse/" + url.PathEscape(issue.Key),
				"summary":  issue.Fields.Summary,
				"status":   issue.Fields.Status.Name,
				"priority": issue.Fields.Priority.Name,
				"assignee": assignee,
				"updated":  normalizeJiraTimestamp(issue.Fields.Updated),
			},
		})
	}

	return detailReportResponse{
		Title: "Jira Issues",
		Columns: []detailReportColumn{
			{Key: "key", Label: "Key", Kind: "external_link", URLKey: "url"},
			{Key: "summary", Label: "Summary"},
			{Key: "status", Label: "Status"},
			{Key: "priority", Label: "Priority"},
			{Key: "assignee", Label: "Assignee"},
			{Key: "updated", Label: "Updated"},
		},
		Rows: rows,
	}, nil
}

func expandJiraTemplate(template string, params detailReportParams) string {
	result := strings.TrimSpace(template)
	result = strings.ReplaceAll(result, "{value}", params.Value)
	result = strings.ReplaceAll(result, "{field}", params.Key)
	result = strings.ReplaceAll(result, "{row}", params.RowJSON)
	return result
}

func normalizeJiraTimestamp(value string) string {
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format("2006-01-02 15:04:05")
	}
	if parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", value); err == nil {
		return parsed.Format("2006-01-02 15:04:05")
	}
	return value
}

var propertiesTemplatePattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::([^}]*))?\}`)

func executePropertiesTemplateReport(ctx context.Context, containerService *container_support.ContainerService, report cardreports.Definition) (detailReportResponse, error) {
	var stdout bytes.Buffer
	if err := containerService.Exec(ctx, []string{"cat", report.FilePath}, &closedExecEventReader{}, &stdout); err != nil {
		return detailReportResponse{}, err
	}

	rows := parsePropertiesTemplateReport(stdout.String(), containerService.Container.Env, report.ExcludePatterns)
	return detailReportResponse{
		Title: "Properties",
		Columns: []detailReportColumn{
			{Key: "parameter", Label: "parameter"},
			{Key: "source", Label: "source"},
			{Key: "value", Label: "value"},
		},
		Rows: rows,
	}, nil
}

func executeFileListingReport(ctx context.Context, containerService *container_support.ContainerService, report cardreports.Definition) (detailReportResponse, error) {
	var stdout bytes.Buffer
	sizeFlag := "0"
	checksumFlag := "0"
	if report.ShowFileSize {
		sizeFlag = "1"
	}
	if report.ShowChecksum {
		checksumFlag = "1"
	}
	execArgs := []string{
		"sh",
		"-lc",
		`found=0;
for file in "$1"/$2; do
	if [ -f "$file" ]; then
		ts=""
		size=""
		checksum=""
		if command -v stat >/dev/null 2>&1; then
			ts="$(stat -c "%y" "$file" 2>/dev/null || true)"
			if [ "$3" = "1" ]; then
				size="$(stat -c "%s" "$file" 2>/dev/null || true)"
			fi
		fi
		if [ "$3" = "1" ] && [ -z "$size" ] && command -v wc >/dev/null 2>&1; then
			size="$(wc -c < "$file" 2>/dev/null | tr -d '[:space:]')"
		fi
		if [ "$4" = "1" ]; then
			if command -v sha256sum >/dev/null 2>&1; then
				checksum="$(sha256sum "$file" 2>/dev/null | awk '{print $1}')"
			elif command -v shasum >/dev/null 2>&1; then
				checksum="$(shasum -a 256 "$file" 2>/dev/null | awk '{print $1}')"
			fi
		fi
		printf "%s|%s|%s|%s\n" "$ts" "$size" "$checksum" "$file"
		found=1
	fi
done
if [ "$found" -eq 0 ] && command -v find >/dev/null 2>&1; then
	find "$1" -type f -name "$2" -print 2>/dev/null | while IFS= read -r file; do
		ts=""
		size=""
		checksum=""
		if command -v stat >/dev/null 2>&1; then
			ts="$(stat -c "%y" "$file" 2>/dev/null || true)"
			if [ "$3" = "1" ]; then
				size="$(stat -c "%s" "$file" 2>/dev/null || true)"
			fi
		fi
		if [ "$3" = "1" ] && [ -z "$size" ] && command -v wc >/dev/null 2>&1; then
			size="$(wc -c < "$file" 2>/dev/null | tr -d '[:space:]')"
		fi
		if [ "$4" = "1" ]; then
			if command -v sha256sum >/dev/null 2>&1; then
				checksum="$(sha256sum "$file" 2>/dev/null | awk '{print $1}')"
			elif command -v shasum >/dev/null 2>&1; then
				checksum="$(shasum -a 256 "$file" 2>/dev/null | awk '{print $1}')"
			fi
		fi
		printf "%s|%s|%s|%s\n" "$ts" "$size" "$checksum" "$file"
	done
fi`,
		"dozzle-file-listing",
		report.FilePath,
		report.FilePattern,
		sizeFlag,
		checksumFlag,
	}
	log.Debug().
		Str("container", containerService.Container.Name).
		Str("containerID", containerService.Container.ID).
		Str("path", report.FilePath).
		Str("pattern", report.FilePattern).
		Msg("running file listing report")
	if err := containerService.Exec(ctx, execArgs, &closedExecEventReader{}, &stdout); err != nil {
		return detailReportResponse{}, err
	}

	output := stdout.String()
	rows := parseFileListingRows(output, report.ExcludePatterns, report.LinkedReportID != "", report.ShowFileSize, report.ShowChecksum)
	log.Debug().
		Str("container", containerService.Container.Name).
		Str("path", report.FilePath).
		Str("pattern", report.FilePattern).
		Int("stdoutBytes", len(output)).
		Int("rowCount", len(rows)).
		Str("preview", previewLogValue(output, 400)).
		Msg("file listing report finished")
	if len(rows) == 0 {
		trimmed := strings.TrimSpace(output)
		log.Warn().
			Str("container", containerService.Container.Name).
			Str("path", report.FilePath).
			Str("pattern", report.FilePattern).
			Str("preview", previewLogValue(trimmed, 400)).
			Msg("file listing report returned no rows")
		if trimmed != "" {
			return detailReportResponse{}, fmt.Errorf("file listing command returned no parseable rows: %s", firstLine(trimmed))
		}
	}

	return detailReportResponse{
		Title:   report.Name,
		Type:    "table",
		Columns: buildFileListingColumns(report),
		Rows:    rows,
	}, nil
}

func executeFileContentReport(ctx context.Context, containerService *container_support.ContainerService, report cardreports.Definition) (detailReportResponse, error) {
	if !isPreviewableTextFile(report.FilePath) {
		return detailReportResponse{}, fmt.Errorf("file preview is not supported for this file type")
	}

	var stdout bytes.Buffer
	maxBytes := filePreviewMaxBytes()
	execArgs := []string{"head", "-c", strconv.Itoa(maxBytes + 1), report.FilePath}
	if err := containerService.Exec(ctx, execArgs, &closedExecEventReader{}, &stdout); err != nil {
		return detailReportResponse{}, err
	}

	content := stdout.Bytes()
	if len(content) > maxBytes {
		return detailReportResponse{}, fmt.Errorf("file is too large to preview")
	}
	if bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
		return detailReportResponse{}, fmt.Errorf("file preview is only supported for text files")
	}

	return detailReportResponse{
		Title:  report.Name,
		Type:   "text",
		Format: detectFileContentFormat(report.FilePath, content),
		Text:   string(content),
	}, nil
}

func parsePropertiesTemplateReport(content string, env []string, excludePatterns []string) []detailReportRow {
	envMap := make(map[string]string, len(env))
	for _, item := range env {
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		envMap[key] = value
	}

	rows := make([]detailReportRow, 0)
	for _, entry := range parsePropertiesEntries(content) {
		if entry.Key == "" || matchesAnyPattern(entry.Key, excludePatterns) {
			continue
		}

		value := entry.Value
		source := "literal"
		if templateMatches := propertiesTemplatePattern.FindAllStringSubmatch(value, -1); len(templateMatches) > 0 {
			hasEnv := false
			hasDefault := false
			value = propertiesTemplatePattern.ReplaceAllStringFunc(value, func(match string) string {
				matches := propertiesTemplatePattern.FindStringSubmatch(match)
				if len(matches) < 2 {
					return match
				}
				if resolved, ok := envMap[matches[1]]; ok && resolved != "" {
					hasEnv = true
					return resolved
				}
				if len(matches) >= 3 {
					hasDefault = true
					return matches[2]
				}
				return ""
			})
			switch {
			case hasEnv:
				source = "env"
			case hasDefault:
				source = "default"
			default:
				source = "template"
			}
		}
		rows = append(rows, detailReportRow{
			Values: map[string]string{
				"parameter": entry.Key,
				"source":    source,
				"value":     value,
			},
		})
	}

	return rows
}

type propertiesEntry struct {
	Key   string
	Value string
}

func parsePropertiesEntries(content string) []propertiesEntry {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	entries := make([]propertiesEntry, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		for hasTrailingContinuation(line) && i+1 < len(lines) {
			nextLine := strings.TrimLeft(lines[i+1], " \t\f")
			line = stripTrailingContinuation(line) + nextLine
			i++
		}

		key, value, ok := parsePropertiesLine(line)
		if !ok {
			continue
		}
		entries = append(entries, propertiesEntry{
			Key:   unescapePropertiesText(key),
			Value: unescapePropertiesText(value),
		})
	}

	return entries
}

func parsePropertiesLine(line string) (string, string, bool) {
	trimmed := strings.TrimLeft(line, " \t\f")
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
		return "", "", false
	}

	separator := -1
	escaped := false
	for idx, r := range trimmed {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '=' || r == ':' || r == ' ' || r == '\t' || r == '\f' {
			separator = idx
			break
		}
	}

	if separator == -1 {
		return strings.TrimSpace(trimmed), "", true
	}

	key := strings.TrimRight(trimmed[:separator], " \t\f")
	valueStart := separator
	for valueStart < len(trimmed) {
		switch trimmed[valueStart] {
		case ' ', '\t', '\f':
			valueStart++
			continue
		case '=', ':':
			valueStart++
			for valueStart < len(trimmed) && (trimmed[valueStart] == ' ' || trimmed[valueStart] == '\t' || trimmed[valueStart] == '\f') {
				valueStart++
			}
		}
		break
	}

	return key, trimmed[valueStart:], true
}

func hasTrailingContinuation(line string) bool {
	backslashes := 0
	for i := len(line) - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func stripTrailingContinuation(line string) string {
	if line == "" {
		return line
	}
	return line[:len(line)-1]
}

func unescapePropertiesText(value string) string {
	if value == "" || !strings.ContainsRune(value, '\\') {
		return value
	}

	var out strings.Builder
	out.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			out.WriteByte(value[i])
			continue
		}

		i++
		switch value[i] {
		case 't':
			out.WriteByte('\t')
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 'f':
			out.WriteByte('\f')
		case '\\', ':', '=', ' ', '#', '!':
			out.WriteByte(value[i])
		case 'u':
			if i+4 < len(value) {
				if r, ok := decodeUnicodeEscape(value[i+1 : i+5]); ok {
					out.WriteRune(r)
					i += 4
					break
				}
			}
			out.WriteString(`\u`)
		default:
			out.WriteByte(value[i])
		}
	}

	return out.String()
}

func decodeUnicodeEscape(raw string) (rune, bool) {
	var value rune
	for _, char := range raw {
		value <<= 4
		switch {
		case char >= '0' && char <= '9':
			value += char - '0'
		case char >= 'a' && char <= 'f':
			value += char - 'a' + 10
		case char >= 'A' && char <= 'F':
			value += char - 'A' + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func buildFileListingColumns(report cardreports.Definition) []detailReportColumn {
	columns := []detailReportColumn{
		{Key: "updated_at", Label: "updated_at"},
		{Key: "file", Label: "file"},
	}
	if report.ShowFileSize {
		columns = append(columns, detailReportColumn{Key: "size", Label: "size"})
	}
	if report.ShowChecksum {
		columns = append(columns, detailReportColumn{Key: "checksum", Label: "sha256"})
	}
	if report.LinkedReportID != "" {
		columns = append(columns, detailReportColumn{
			Key:         "linked_report",
			Label:       firstNonEmpty(report.LinkedReportLabel, "report"),
			Kind:        "report_link",
			ReportID:    report.LinkedReportID,
			ParamKey:    "file",
			ButtonLabel: firstNonEmpty(report.LinkedReportLabel, "Open report"),
			PassRow:     true,
		})
	}
	return columns
}

func parseFileListingRows(output string, excludePatterns []string, includeLinkedReport bool, includeSize bool, includeChecksum bool) []detailReportRow {
	rows := make([]detailReportRow, 0)
	seen := make(map[string]struct{})
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		updatedAt := ""
		size := ""
		checksum := ""
		filePath := line
		switch parts := strings.SplitN(line, "|", 4); len(parts) {
		case 4:
			updatedAt = normalizeFileTimestamp(parts[0])
			size = formatFileSize(parts[1])
			checksum = strings.TrimSpace(parts[2])
			filePath = strings.TrimSpace(parts[3])
		case 2:
			updatedAt = normalizeFileTimestamp(parts[0])
			filePath = strings.TrimSpace(parts[1])
		}

		fileName := filepath.Base(strings.TrimSpace(filePath))
		if fileName == "" || matchesAnyPattern(fileName, excludePatterns) {
			continue
		}
		if _, ok := seen[fileName]; ok {
			continue
		}
		seen[fileName] = struct{}{}

		values := map[string]string{
			"updated_at": updatedAt,
			"file":       fileName,
		}
		if includeSize {
			values["size"] = size
		}
		if includeChecksum {
			values["checksum"] = checksum
		}
		if includeLinkedReport {
			values["linked_report"] = "open"
		}
		rows = append(rows, detailReportRow{Values: values})
	}
	return rows
}

func firstLine(value string) string {
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		return value[:index]
	}
	return value
}

func previewLogValue(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func normalizeFileTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 19 {
		return value[:19]
	}
	return value
}

func formatFileSize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	size, err := strconv.ParseInt(value, 10, 64)
	if err != nil || size < 0 {
		return value
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	display := float64(size)
	for _, unit := range units {
		display /= 1024
		if display < 1024 {
			if display >= 10 {
				return fmt.Sprintf("%.0f %s", display, unit)
			}
			return fmt.Sprintf("%.1f %s", display, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", display/1024)
}

func isPreviewableTextFile(filePath string) bool {
	name := strings.ToLower(strings.TrimSpace(filePath))
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	switch {
	case strings.HasSuffix(name, ".log"),
		strings.HasSuffix(name, ".gz"),
		strings.HasSuffix(name, ".zip"),
		strings.HasSuffix(name, ".jar"),
		strings.HasSuffix(name, ".war"),
		strings.HasSuffix(name, ".class"),
		strings.HasSuffix(name, ".so"),
		strings.HasSuffix(name, ".png"),
		strings.HasSuffix(name, ".jpg"),
		strings.HasSuffix(name, ".jpeg"),
		strings.HasSuffix(name, ".gif"),
		strings.HasSuffix(name, ".pdf"):
		return false
	}

	allowed := filePreviewAllowedFormats()
	if ext == "" {
		_, ok := allowed["text"]
		return ok
	}

	_, ok := allowed[ext]
	return ok
}

func detectFileContentFormat(filePath string, content []byte) string {
	name := strings.ToLower(strings.TrimSpace(filePath))
	switch {
	case strings.HasSuffix(name, ".json"):
		return "json"
	case strings.HasSuffix(name, ".yaml"), strings.HasSuffix(name, ".yml"):
		return "yaml"
	case strings.HasSuffix(name, ".xml"):
		return "xml"
	case strings.HasSuffix(name, ".html"), strings.HasSuffix(name, ".htm"):
		return "html"
	}

	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return "text"
	}
	if json.Valid(trimmed) {
		return "json"
	}

	lower := strings.ToLower(string(trimmed))
	switch {
	case strings.HasPrefix(lower, "<!doctype html"), strings.HasPrefix(lower, "<html"):
		return "html"
	case strings.HasPrefix(lower, "<?xml"), strings.HasPrefix(lower, "<"):
		return "xml"
	case bytes.Contains(trimmed, []byte(":")):
		return "yaml"
	default:
		return "text"
	}
}

func filePreviewAllowedFormats() map[string]struct{} {
	formats := []string{
		"json", "yaml", "yml", "xml", "html", "htm",
		"txt", "text", "conf", "cfg", "ini", "properties", "md",
	}
	if custom := strings.TrimSpace(os.Getenv(filePreviewFormatsEnvVar)); custom != "" {
		formats = formats[:0]
		for _, item := range strings.Split(custom, ",") {
			item = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(item, ".")))
			if item == "" {
				continue
			}
			formats = append(formats, item)
		}
	}

	allowed := make(map[string]struct{}, len(formats))
	for _, format := range formats {
		allowed[format] = struct{}{}
	}
	return allowed
}

func filePreviewMaxBytes() int {
	raw := strings.TrimSpace(os.Getenv(filePreviewMaxBytesEnvVar))
	if raw == "" {
		return defaultMaxFileContentBytes
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultMaxFileContentBytes
	}
	return value
}

func matchesAnyPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if ok, _ := path.Match(pattern, value); ok {
			return true
		}
	}
	return false
}

func stringifyReportValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func detailReportCacheKey(report cardreports.Definition, host string, c container.Container, params detailReportParams) string {
	paramSuffix := ""
	if params.Key != "" || params.Value != "" || params.RowJSON != "" {
		paramSuffix = ":field=" + params.Key + ":key=" + params.Value + ":row=" + params.RowJSON
	}
	if report.RefreshMode == cardreports.RefreshModeContainerUpdate {
		return "event:" + detailReportStableContainerKey(host, c) + ":" + report.ID + paramSuffix
	}
	if report.RefreshMode == cardreports.RefreshModeManual {
		return "manual:" + host + ":" + c.ID + ":" + report.ID + paramSuffix
	}
	return "ttl:" + host + ":" + c.ID + ":" + report.ID + paramSuffix
}

func detailReportStableContainerKey(host string, c container.Container) string {
	if alias := strings.TrimSpace(c.Labels[containerReportIdentityLabel]); alias != "" {
		return host + ":label:" + alias
	}
	return host + ":name:" + c.Name
}
