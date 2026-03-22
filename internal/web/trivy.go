package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/trivy"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type TrivyScanner interface {
	ScanImage(ctx context.Context, image string) (*trivy.Result, error)
}

func (h *handler) scanContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
		log.Warn().Msg("user is not permitted to scan container")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	if h.trivyScanner == nil {
		http.Error(w, "trivy scanner is not configured", http.StatusServiceUnavailable)
		return
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	result, err := h.trivyScanner.ScanImage(r.Context(), containerService.Container.Image)
	if err != nil {
		log.Error().Err(err).Str("image", containerService.Container.Image).Msg("error while scanning image with trivy")

		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, trivy.ErrImageRequired):
			status = http.StatusBadRequest
		case errors.Is(err, exec.ErrNotFound):
			status = http.StatusServiceUnavailable
		}

		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Error().Err(err).Msg("error while encoding trivy scan response")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
