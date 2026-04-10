package web

import (
	"context"
	"net/url"
	"strings"

	"github.com/amir20/dozzle/internal/cardtemplates"
	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

func (h *handler) startContainerCardAutoInjection() {
	if !h.config.EnableActions || h.cardTemplateManager == nil || strings.TrimSpace(h.config.PublicURL) == "" {
		return
	}

	containers := make(chan container.Container, 32)
	h.hostService.SubscribeContainersStarted(context.Background(), containers, func(*container.Container) bool { return true })

	go func() {
		for c := range containers {
			if _, loaded := h.autoInjected.LoadOrStore(c.Host+":"+c.ID, struct{}{}); loaded {
				continue
			}
			go h.autoInjectContainerCard(c)
		}
	}()
}

func (h *handler) autoInjectContainerCard(c container.Container) {
	template := cardtemplates.ResolveTemplateForContainer(h.cardTemplateManager.Templates(), c)
	if template == nil || !template.ShowActions {
		return
	}

	hasInjectAction := false
	for _, action := range template.Actions {
		if action == "inject-logs-button" {
			hasInjectAction = true
			break
		}
	}
	if !hasInjectAction {
		return
	}
	if strings.TrimSpace(template.InjectIndexPath) == "" {
		return
	}

	alias := resolveTemplateAlias(c, template.InjectAliasSource)
	if alias == "" {
		log.Debug().Str("container", c.Name).Msg("skipping auto log button injection because alias is empty")
		return
	}

	logsURL, err := buildPublicContainerRefURL(h.config.PublicURL, h.config.Base, alias)
	if err != nil {
		log.Warn().Err(err).Str("container", c.Name).Msg("skipping auto log button injection because public URL is invalid")
		return
	}

	containerService, err := h.hostService.FindContainer(c.Host, c.ID, nil)
	if err != nil {
		log.Warn().Err(err).Str("container", c.Name).Msg("unable to find container for auto log button injection")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultActionTimeout)
	defer cancel()

	if err := executeInjectLogsButton(ctx, containerService, injectLogsButtonRequest{
		IndexPath: template.InjectIndexPath,
		Alias:     alias,
		LogsURL:   logsURL,
	}); err != nil {
		log.Warn().Err(err).Str("container", c.Name).Msg("automatic log button injection failed")
		return
	}

	log.Info().Str("container", c.Name).Str("alias", alias).Msg("automatic log button injection completed")
}

func resolveTemplateAlias(c container.Container, configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return ""
	}
	if configured == "name" || configured == "id" || strings.HasPrefix(configured, "label:") {
		return strings.TrimSpace(cardtemplates.ResolveContainerTextSource(c, configured))
	}
	return configured
}

func buildPublicContainerRefURL(publicURL, base, alias string) (string, error) {
	root := strings.TrimRight(strings.TrimSpace(publicURL), "/")
	base = "/" + strings.Trim(strings.TrimSpace(base), "/")
	if base == "/" {
		base = ""
	}

	_, err := url.ParseRequestURI(root)
	if err != nil {
		return "", err
	}

	return root + base + "/container/ref/" + url.PathEscape(alias), nil
}
