package cardtemplates

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
)

func TestMatchesContainer(t *testing.T) {
	c := container.Container{
		ID:      "123",
		Name:    "udg-backend-1",
		Image:   "repo/backend:v2",
		Host:    "prod-host",
		State:   "running",
		Command: "java -jar app.jar",
		Labels: map[string]string{
			"com.docker.compose.project": "udg",
		},
	}

	assert.True(t, MatchesContainer(c, `name contains "backend" && host == "prod-host"`))
	assert.True(t, MatchesContainer(c, `namespace == "udg"`))
	assert.False(t, MatchesContainer(c, `image contains "frontend"`))
}

func TestResolveTemplateForContainer(t *testing.T) {
	templates := []Template{
		{ID: "default", Name: "Default", Enabled: true},
		{ID: "backend", Name: "Backend", Enabled: true, Filter: `name contains "backend"`},
	}
	c := container.Container{Name: "udg-backend-1"}

	template := ResolveTemplateForContainer(templates, c)
	if assert.NotNil(t, template) {
		assert.Equal(t, "default", template.ID)
	}
}
