package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type logIncidentMatchRequest struct {
	LogMsg string `json:"logMsg"`
}

func (h *handler) matchLogIncident(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.config.LogIncidentEndpoint) == "" {
		writeError(w, http.StatusNotFound, "log incident endpoint is not configured")
		return
	}

	userLabels := h.config.Labels
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), chi.URLParam(r, "id"), userLabels)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var input logIncidentMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.LogMsg = strings.TrimSpace(input.LogMsg)
	if input.LogMsg == "" {
		writeError(w, http.StatusBadRequest, "log message is required")
		return
	}

	payload, err := json.Marshal(map[string]string{
		"container": containerService.Container.Name,
		"log_msg":   input.LogMsg,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode request")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimSpace(h.config.LogIncidentEndpoint), bytes.NewReader(payload))
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to build incident match request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("url", h.config.LogIncidentEndpoint).Msg("incident match request failed")
		writeError(w, http.StatusBadGateway, "failed to fetch incident match")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("incident match endpoint returned %d", resp.StatusCode))
		return
	}

	var output any
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		writeError(w, http.StatusBadGateway, "invalid incident match response")
		return
	}
	writeJSON(w, http.StatusOK, output)
}
