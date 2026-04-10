package cardtemplates

import (
	"strings"

	"github.com/amir20/dozzle/internal/container"
)

func ResolveContainerTextSource(c container.Container, source string) string {
	switch source {
	case "name":
		return c.Name
	case "image":
		return c.Image
	case "command":
		return c.Command
	case "host":
		return c.Host
	case "state":
		return c.State
	case "group":
		return c.Group
	case "namespace":
		return resolveNamespace(c)
	case "health":
		return c.Health
	case "id":
		return c.ID
	default:
		if strings.HasPrefix(source, "label:") {
			return c.Labels[strings.TrimPrefix(source, "label:")]
		}
		return ""
	}
}

func resolveNamespace(c container.Container) string {
	return firstNonEmpty(
		c.Labels["dev.dozzle.group"],
		c.Labels["coolify.projectName"],
		c.Labels["com.docker.stack.namespace"],
		c.Labels["com.docker.compose.project"],
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func MatchesContainer(c container.Container, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}

	for _, part := range strings.Split(filter, "&&") {
		if !evaluateCondition(c, strings.TrimSpace(part)) {
			return false
		}
	}

	return true
}

func evaluateCondition(c container.Container, raw string) bool {
	if raw == "" {
		return true
	}

	matchOp := "contains"
	idx := strings.Index(raw, " contains ")
	if idx == -1 {
		matchOp = "=="
		idx = strings.Index(raw, " == ")
	}
	if idx == -1 {
		return false
	}

	field := strings.TrimSpace(raw[:idx])
	value := strings.TrimSpace(raw[idx+len(matchOp)+2:])
	value = stripQuotes(value)
	left := ResolveContainerTextSource(c, field)

	if matchOp == "==" {
		return left == value
	}
	return strings.Contains(strings.ToLower(left), strings.ToLower(value))
}

func stripQuotes(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func ResolveTemplateForContainer(templates []Template, c container.Container) *Template {
	for _, template := range templates {
		if template.Enabled && MatchesContainer(c, template.Filter) {
			copyTemplate := template
			return &copyTemplate
		}
	}
	return nil
}
