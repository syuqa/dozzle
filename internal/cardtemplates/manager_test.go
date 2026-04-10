package cardtemplates

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager_PersistsTemplates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "container_card_templates.json")

	manager, err := NewManagerWithPath(path)
	require.NoError(t, err)

	saved, err := manager.ReplaceTemplates([]Template{
		{
			ID:          "backend",
			Name:        "Backend",
			Enabled:     true,
			Filter:      `name contains "backend"`,
			ShowHost:    true,
			ShowState:   true,
			ShowCreated: false,
			ExtraFields: []Field{
				{ID: "field-service", Label: "Service", Source: "label:com.docker.compose.service"},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, saved, 1)

	reloaded, err := NewManagerWithPath(path)
	require.NoError(t, err)

	templates := reloaded.Templates()
	require.Len(t, templates, 1)
	require.Equal(t, "backend", templates[0].ID)
	require.Equal(t, "Service", templates[0].ExtraFields[0].Label)
}

func TestManager_DefaultTemplateWhenEmpty(t *testing.T) {
	manager, err := NewManagerWithPath(filepath.Join(t.TempDir(), "container_card_templates.json"))
	require.NoError(t, err)

	templates, err := manager.ReplaceTemplates(nil)
	require.NoError(t, err)
	require.Len(t, templates, 1)
	require.Equal(t, "default", templates[0].ID)
}
