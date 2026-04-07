package dispatcher

import (
	"bytes"
	"fmt"
	"text/template"
)

type TemplateOverrideCapable interface {
	WithTemplate(templateText string) (Dispatcher, error)
}

func executeTextTemplate(templateText string, data any) (string, error) {
	tmpl, err := template.New("notification").Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}
