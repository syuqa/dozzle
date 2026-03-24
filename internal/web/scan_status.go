package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) getContainerScanStatus(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.config.ScanStatusEndpoint) == "" {
		writeError(w, http.StatusNotFound, "scan status endpoint is not configured")
		return
	}

	userLabels := h.config.Labels
	permit := true
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
		permit = user.Roles.Has(auth.Actions)
	}

	if !permit {
		writeError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), chi.URLParam(r, "id"), userLabels)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	endpoint, err := url.Parse(strings.TrimSpace(h.config.ScanStatusEndpoint))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "invalid scan status endpoint")
		return
	}

	query := endpoint.Query()
	query.Set("container", containerService.Container.Name)
	query.Set("image", containerService.Container.Image)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to build scan status request")
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("url", endpoint.String()).Msg("scan status request failed")
		writeError(w, http.StatusBadGateway, "failed to fetch scan status")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("scan status endpoint returned %d", resp.StatusCode))
		return
	}

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadGateway, "invalid scan status response")
		return
	}
	writeJSON(w, http.StatusOK, normalizeScanStatusPayload(payload))
}

func normalizeScanStatusPayload(payload any) any {
	root, ok := payload.(map[string]any)
	if !ok {
		return payload
	}

	normalizeGroup := func(value any) {
		group, ok := value.(map[string]any)
		if !ok {
			return
		}
		for _, key := range []string{"open", "resolved", "false_positive"} {
			items, ok := group[key].([]any)
			if !ok {
				continue
			}
			for _, item := range items {
				issue, ok := item.(map[string]any)
				if !ok {
					continue
				}
				renameKey(issue, "status_label", "statusLabel")
				renameKey(issue, "created_at", "createdAt")
				renameKey(issue, "updated_at", "updatedAt")
			}
		}
		renameKey(group, "false_positive", "falsePositive")
	}

	if incidents, ok := root["incidents"]; ok {
		normalizeGroup(incidents)
	}
	if cve, ok := root["cve"]; ok {
		normalizeGroup(cve)
	}
	renameKey(root, "overall_status", "overallStatus")
	renameKey(root, "gitlab_url", "gitlabUrl")
	return root
}

func renameKey(m map[string]any, from, to string) {
	if value, ok := m[from]; ok {
		m[to] = value
		delete(m, from)
	}
}
