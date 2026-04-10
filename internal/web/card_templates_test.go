package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/cardtemplates"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestCardTemplatesAPI(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644))

	manager, err := cardtemplates.NewManagerWithPath(t.TempDir() + "/container_card_templates.json")
	require.NoError(t, err)

	router := createRouter(&handler{
		content:             afero.NewIOFS(fs),
		config:              &Config{Base: "/", Authorization: Authorization{Provider: NONE}},
		cardTemplateManager: manager,
	})

	update := []cardtemplates.Template{
		{
			ID:          "backend",
			Name:        "Backend",
			Enabled:     true,
			Filter:      `name contains "backend"`,
			ShowHost:    true,
			ShowState:   true,
			ShowCreated: true,
			ExtraFields: []cardtemplates.Field{
				{ID: "field-service", Label: "Service", Source: "label:com.docker.compose.service"},
			},
		},
	}
	body, err := json.Marshal(update)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/card-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/card-templates", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var got []cardtemplates.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
	require.Equal(t, "backend", got[0].ID)
	require.Equal(t, "Service", got[0].ExtraFields[0].Label)
}
