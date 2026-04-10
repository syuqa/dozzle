package cardtemplates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const defaultConfigPath = "./data/container_card_templates.json"

type FieldSource string
type FieldKind string

const (
	FieldKindText         FieldKind = "text"
	FieldKindTemplateText FieldKind = "template_text"
	FieldKindStaticText   FieldKind = "static_text"
	FieldKindLink         FieldKind = "link"
	FieldKindReport       FieldKind = "report"
)

type Field struct {
	ID        string      `json:"id"`
	Label     string      `json:"label"`
	Kind      FieldKind   `json:"kind,omitempty"`
	Source    FieldSource `json:"source,omitempty"`
	TextValue string      `json:"textValue,omitempty"`
	LinkURL   string      `json:"linkUrl,omitempty"`
	LinkText  string      `json:"linkText,omitempty"`
	ReportID  string      `json:"reportId,omitempty"`
}

type DetailGroup struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

type Template struct {
	ID                        string        `json:"id"`
	Name                      string        `json:"name"`
	IconURL                   string        `json:"iconUrl,omitempty"`
	Enabled                   bool          `json:"enabled"`
	Filter                    string        `json:"filter"`
	ShowHost                  bool          `json:"showHost"`
	ShowState                 bool          `json:"showState"`
	ShowCreated               bool          `json:"showCreated"`
	ShowDetails               bool          `json:"showDetails"`
	ShowActions               bool          `json:"showActions"`
	Actions                   []string      `json:"actions"`
	InjectIndexPath           string        `json:"injectIndexPath,omitempty"`
	InjectAliasSource         string        `json:"injectAliasSource,omitempty"`
	ShowVulnerabilities       bool          `json:"showVulnerabilities,omitempty"`
	VulnerabilityPackageTypes []string      `json:"vulnerabilityPackageTypes,omitempty"`
	VulnerabilitySeverities   []string      `json:"vulnerabilitySeverities,omitempty"`
	ExtraFields               []Field       `json:"extraFields"`
	DetailGroups              []DetailGroup `json:"detailGroups,omitempty"`
	DetailFields              []Field       `json:"detailFields"`
}

type Manager struct {
	path      string
	mu        sync.RWMutex
	templates []Template
}

func NewManager() (*Manager, error) {
	return NewManagerWithPath(defaultConfigPath)
}

func NewManagerWithPath(path string) (*Manager, error) {
	m := &Manager{
		path:      path,
		templates: []Template{defaultTemplate()},
	}

	if err := m.load(); err != nil {
		return nil, err
	}

	if len(m.templates) == 0 {
		m.templates = []Template{defaultTemplate()}
		if err := m.saveLocked(); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *Manager) Templates() []Template {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return cloneTemplates(m.templates)
}

func (m *Manager) ReplaceTemplates(next []Template) ([]Template, error) {
	sanitized, err := sanitizeTemplates(next)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.templates = sanitized
	if err := m.saveLocked(); err != nil {
		return nil, err
	}

	return cloneTemplates(m.templates), nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open container card templates: %w", err)
	}
	defer file.Close()

	var stored []Template
	if err := json.NewDecoder(file).Decode(&stored); err != nil {
		return fmt.Errorf("decode container card templates: %w", err)
	}

	sanitized, err := sanitizeTemplates(stored)
	if err != nil {
		return err
	}

	m.templates = sanitized
	return nil
}

func (m *Manager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return fmt.Errorf("create card templates data dir: %w", err)
	}

	file, err := os.Create(m.path)
	if err != nil {
		return fmt.Errorf("create container card templates file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(m.templates); err != nil {
		return fmt.Errorf("write container card templates: %w", err)
	}

	return nil
}

func sanitizeTemplates(next []Template) ([]Template, error) {
	if len(next) == 0 {
		return []Template{defaultTemplate()}, nil
	}

	sanitized := make([]Template, 0, len(next))
	ids := make(map[string]struct{}, len(next))

	for index, tmpl := range next {
		tmpl.ID = strings.TrimSpace(tmpl.ID)
		tmpl.Name = strings.TrimSpace(tmpl.Name)
		tmpl.IconURL = strings.TrimSpace(tmpl.IconURL)
		tmpl.Filter = strings.TrimSpace(tmpl.Filter)
		tmpl.Actions = sanitizeActions(tmpl.Actions)
		tmpl.InjectIndexPath = strings.TrimSpace(tmpl.InjectIndexPath)
		tmpl.InjectAliasSource = strings.TrimSpace(tmpl.InjectAliasSource)
		tmpl.VulnerabilityPackageTypes = sanitizeStringList(tmpl.VulnerabilityPackageTypes)
		tmpl.VulnerabilitySeverities = sanitizeStringList(tmpl.VulnerabilitySeverities)

		if tmpl.ID == "" {
			return nil, fmt.Errorf("template %d is missing id", index)
		}
		if _, exists := ids[tmpl.ID]; exists {
			return nil, fmt.Errorf("duplicate template id %q", tmpl.ID)
		}
		ids[tmpl.ID] = struct{}{}

		if tmpl.Name == "" {
			return nil, fmt.Errorf("template %q is missing name", tmpl.ID)
		}

		extraFields, err := sanitizeFields(tmpl.ID, tmpl.ExtraFields)
		if err != nil {
			return nil, err
		}
		detailGroups, err := sanitizeDetailGroups(tmpl.ID, tmpl.DetailGroups)
		if err != nil {
			return nil, err
		}
		if len(detailGroups) == 0 {
			detailFields, err := sanitizeFields(tmpl.ID, tmpl.DetailFields)
			if err != nil {
				return nil, err
			}
			if len(detailFields) > 0 {
				detailGroups = []DetailGroup{{ID: "general", Name: "General", Fields: detailFields}}
			}
		}
		if len(detailGroups) == 0 {
			detailGroups = []DetailGroup{{ID: "general", Name: "General", Fields: nil}}
		}

		tmpl.ExtraFields = extraFields
		tmpl.DetailGroups = detailGroups
		tmpl.DetailFields = nil
		sanitized = append(sanitized, tmpl)
	}

	return sanitized, nil
}

func cloneTemplates(source []Template) []Template {
	out := make([]Template, 0, len(source))
	for _, tmpl := range source {
		copyTemplate := tmpl
		copyTemplate.ExtraFields = append([]Field(nil), tmpl.ExtraFields...)
		copyTemplate.DetailGroups = cloneDetailGroups(tmpl.DetailGroups)
		copyTemplate.DetailFields = append([]Field(nil), tmpl.DetailFields...)
		out = append(out, copyTemplate)
	}
	return out
}

func defaultTemplate() Template {
	return Template{
		ID:                        "default",
		Name:                      "Default",
		IconURL:                   "",
		Enabled:                   true,
		Filter:                    "",
		ShowHost:                  true,
		ShowState:                 true,
		ShowCreated:               true,
		ShowDetails:               true,
		ShowActions:               false,
		Actions:                   []string{"restart", "stop", "start", "update"},
		InjectIndexPath:           "/usr/share/nginx/html/index.html",
		InjectAliasSource:         "",
		ShowVulnerabilities:       false,
		VulnerabilityPackageTypes: nil,
		VulnerabilitySeverities:   nil,
		ExtraFields: []Field{
			{ID: "field-image", Label: "Image", Source: "image"},
			{ID: "field-namespace", Label: "Namespace", Source: "namespace"},
		},
		DetailGroups: []DetailGroup{
			{
				ID:   "general",
				Name: "General",
				Fields: []Field{
					{ID: "detail-image", Label: "Image", Source: "image"},
					{ID: "detail-command", Label: "Command", Source: "command"},
					{ID: "detail-host", Label: "Host", Source: "host"},
					{ID: "detail-state", Label: "State", Source: "state"},
					{ID: "detail-created", Label: "Created", Source: "created"},
					{ID: "detail-namespace", Label: "Namespace", Source: "namespace"},
				},
			},
		},
	}
}

func sanitizeFields(templateID string, input []Field) ([]Field, error) {
	fieldIDs := make(map[string]struct{}, len(input))
	fields := make([]Field, 0, len(input))
	for _, field := range input {
		field.ID = strings.TrimSpace(field.ID)
		field.Label = strings.TrimSpace(field.Label)
		field.Kind = FieldKind(strings.TrimSpace(string(field.Kind)))
		field.Source = FieldSource(strings.TrimSpace(string(field.Source)))
		field.TextValue = strings.TrimSpace(field.TextValue)
		field.LinkURL = strings.TrimSpace(field.LinkURL)
		field.LinkText = strings.TrimSpace(field.LinkText)
		field.ReportID = strings.TrimSpace(field.ReportID)

		if field.ID == "" {
			return nil, fmt.Errorf("template %q has field with missing id", templateID)
		}
		if _, exists := fieldIDs[field.ID]; exists {
			return nil, fmt.Errorf("template %q has duplicate field id %q", templateID, field.ID)
		}
		fieldIDs[field.ID] = struct{}{}

		if field.Label == "" {
			return nil, fmt.Errorf("template %q has field with missing label", templateID)
		}
		if field.Kind == "" {
			field.Kind = FieldKindText
		}
		switch field.Kind {
		case FieldKindText:
			if field.Source == "" {
				return nil, fmt.Errorf("template %q has field %q with missing source", templateID, field.ID)
			}
		case FieldKindTemplateText, FieldKindStaticText:
			if field.TextValue == "" {
				return nil, fmt.Errorf("template %q has field %q with missing text value", templateID, field.ID)
			}
		case FieldKindLink:
			if field.LinkURL == "" {
				return nil, fmt.Errorf("template %q has link field %q with missing url", templateID, field.ID)
			}
		case FieldKindReport:
			if field.ReportID == "" {
				return nil, fmt.Errorf("template %q has report field %q with missing report id", templateID, field.ID)
			}
		default:
			return nil, fmt.Errorf("template %q has field %q with unsupported kind %q", templateID, field.ID, field.Kind)
		}

		fields = append(fields, field)
	}

	return fields, nil
}

func sanitizeDetailGroups(templateID string, input []DetailGroup) ([]DetailGroup, error) {
	groupIDs := make(map[string]struct{}, len(input))
	groups := make([]DetailGroup, 0, len(input))
	for _, group := range input {
		group.ID = strings.TrimSpace(group.ID)
		group.Name = strings.TrimSpace(group.Name)
		if group.ID == "" {
			return nil, fmt.Errorf("template %q has detail group with missing id", templateID)
		}
		if _, exists := groupIDs[group.ID]; exists {
			return nil, fmt.Errorf("template %q has duplicate detail group id %q", templateID, group.ID)
		}
		groupIDs[group.ID] = struct{}{}
		if group.Name == "" {
			return nil, fmt.Errorf("template %q has detail group %q with missing name", templateID, group.ID)
		}
		fields, err := sanitizeFields(templateID+":"+group.ID, group.Fields)
		if err != nil {
			return nil, err
		}
		group.Fields = fields
		groups = append(groups, group)
	}
	return groups, nil
}

func cloneDetailGroups(source []DetailGroup) []DetailGroup {
	out := make([]DetailGroup, 0, len(source))
	for _, group := range source {
		copyGroup := group
		copyGroup.Fields = append([]Field(nil), group.Fields...)
		out = append(out, copyGroup)
	}
	return out
}

func sanitizeActions(next []string) []string {
	allowed := map[string]struct{}{
		"start":              {},
		"stop":               {},
		"restart":            {},
		"update":             {},
		"inject-logs-button": {},
	}

	seen := make(map[string]struct{}, len(next))
	sanitized := make([]string, 0, len(next))
	for _, action := range next {
		action = strings.TrimSpace(action)
		if _, ok := allowed[action]; !ok {
			continue
		}
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		sanitized = append(sanitized, action)
	}

	return sanitized
}

func sanitizeStringList(next []string) []string {
	seen := make(map[string]struct{}, len(next))
	sanitized := make([]string, 0, len(next))
	for _, value := range next {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized := strings.ToUpper(value)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		sanitized = append(sanitized, value)
	}
	return sanitized
}
