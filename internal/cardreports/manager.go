package cardreports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const defaultConfigPath = "./data/container_card_reports.json"

type Parser string
type RefreshMode string
type DisplayMode string

const (
	ParserText               Parser = "text"
	ParserJSON               Parser = "json"
	ParserJarModules         Parser = "jar_modules"
	ParserJiraIssues         Parser = "jira_issues"
	ParserPropertiesTemplate Parser = "properties_template"
	ParserFileListing        Parser = "file_listing"
	ParserFileContent        Parser = "file_content"

	RefreshModeTTL             RefreshMode = "ttl"
	RefreshModeContainerUpdate RefreshMode = "container_update"
	RefreshModeManual          RefreshMode = "manual"
	RefreshModeOnOpen          RefreshMode = "on_open"

	DisplayModeInline DisplayMode = "inline"
	DisplayModePage   DisplayMode = "page"
)

type Definition struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	Enabled           bool        `json:"enabled"`
	Description       string      `json:"description,omitempty"`
	Command           string      `json:"command"`
	FilePath          string      `json:"filePath,omitempty"`
	FilePattern       string      `json:"filePattern,omitempty"`
	ExcludePatterns   []string    `json:"excludePatterns,omitempty"`
	LinkedReportID    string      `json:"linkedReportId,omitempty"`
	LinkedReportLabel string      `json:"linkedReportLabel,omitempty"`
	ShowFileSize      bool        `json:"showFileSize,omitempty"`
	ShowChecksum      bool        `json:"showChecksum,omitempty"`
	Parser            Parser      `json:"parser"`
	DisplayMode       DisplayMode `json:"displayMode,omitempty"`
	RefreshMode       RefreshMode `json:"refreshMode,omitempty"`
	TimeoutSeconds    int         `json:"timeoutSeconds"`
	CacheTTLSeconds   int         `json:"cacheTtlSeconds"`
}

type Manager struct {
	path    string
	mu      sync.RWMutex
	reports []Definition
}

func NewManager() (*Manager, error) {
	return NewManagerWithPath(defaultConfigPath)
}

func NewManagerWithPath(path string) (*Manager, error) {
	m := &Manager{path: path}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) Reports() []Definition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Definition(nil), m.reports...)
}

func (m *Manager) Get(id string) (Definition, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, report := range m.reports {
		if report.ID == id {
			return report, true
		}
	}
	return Definition{}, false
}

func (m *Manager) ReplaceReports(input []Definition) ([]Definition, error) {
	sanitized, err := sanitizeReports(input)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports = sanitized
	if err := m.saveLocked(); err != nil {
		return nil, err
	}
	return append([]Definition(nil), m.reports...), nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open container card reports: %w", err)
	}
	defer file.Close()

	var stored []Definition
	if err := json.NewDecoder(file).Decode(&stored); err != nil {
		return fmt.Errorf("decode container card reports: %w", err)
	}

	sanitized, err := sanitizeReports(stored)
	if err != nil {
		return err
	}
	m.reports = sanitized
	return nil
}

func (m *Manager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return fmt.Errorf("create card reports data dir: %w", err)
	}
	file, err := os.Create(m.path)
	if err != nil {
		return fmt.Errorf("create container card reports file: %w", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m.reports); err != nil {
		return fmt.Errorf("write container card reports: %w", err)
	}
	return nil
}

func sanitizeReports(input []Definition) ([]Definition, error) {
	out := make([]Definition, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for i, report := range input {
		report.ID = strings.TrimSpace(report.ID)
		report.Name = strings.TrimSpace(report.Name)
		report.Description = strings.TrimSpace(report.Description)
		report.Command = strings.TrimSpace(report.Command)
		report.FilePath = strings.TrimSpace(report.FilePath)
		report.FilePattern = strings.TrimSpace(report.FilePattern)
		report.ExcludePatterns = sanitizeStringList(report.ExcludePatterns)
		report.LinkedReportID = strings.TrimSpace(report.LinkedReportID)
		report.LinkedReportLabel = strings.TrimSpace(report.LinkedReportLabel)
		if report.TimeoutSeconds <= 0 {
			report.TimeoutSeconds = 10
		}
		if report.CacheTTLSeconds < 0 {
			report.CacheTTLSeconds = 0
		}
		if report.Parser == "" {
			report.Parser = ParserText
		}
		if report.DisplayMode == "" {
			report.DisplayMode = DisplayModeInline
		}
		if report.RefreshMode == "" {
			report.RefreshMode = RefreshModeTTL
		}

		if report.ID == "" {
			return nil, fmt.Errorf("report %d is missing id", i)
		}
		if _, ok := seen[report.ID]; ok {
			return nil, fmt.Errorf("duplicate report id %q", report.ID)
		}
		seen[report.ID] = struct{}{}
		if report.Name == "" {
			return nil, fmt.Errorf("report %q is missing name", report.ID)
		}
		switch report.Parser {
		case ParserText, ParserJSON, ParserJarModules, ParserJiraIssues:
			if report.Command == "" {
				return nil, fmt.Errorf("report %q is missing command", report.ID)
			}
		case ParserPropertiesTemplate, ParserFileContent:
			if report.FilePath == "" {
				return nil, fmt.Errorf("report %q is missing file path", report.ID)
			}
		case ParserFileListing:
			if report.FilePath == "" {
				return nil, fmt.Errorf("report %q is missing file path", report.ID)
			}
			if report.FilePattern == "" {
				return nil, fmt.Errorf("report %q is missing file pattern", report.ID)
			}
		default:
			return nil, fmt.Errorf("report %q has unsupported parser %q", report.ID, report.Parser)
		}
		switch report.DisplayMode {
		case DisplayModeInline, DisplayModePage:
		default:
			return nil, fmt.Errorf("report %q has unsupported display mode %q", report.ID, report.DisplayMode)
		}
		switch report.RefreshMode {
		case RefreshModeTTL, RefreshModeContainerUpdate, RefreshModeManual, RefreshModeOnOpen:
		default:
			return nil, fmt.Errorf("report %q has unsupported refresh mode %q", report.ID, report.RefreshMode)
		}

		out = append(out, report)
	}
	return out, nil
}

func sanitizeStringList(input []string) []string {
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, value := range input {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
