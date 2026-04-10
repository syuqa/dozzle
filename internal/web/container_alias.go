package web

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

const containerLinkLabel = "dev.dozzle.link-id"

func (h *handler) redirectContainerByLabel(w http.ResponseWriter, r *http.Request) {
	h.redirectContainerByLabelTo(w, r, "")
}

func (h *handler) redirectContainerDetailsByLabel(w http.ResponseWriter, r *http.Request) {
	h.redirectContainerByLabelTo(w, r, "/details")
}

func (h *handler) redirectContainerByLabelTo(w http.ResponseWriter, r *http.Request, suffix string) {
	ref := strings.TrimSpace(chi.URLParam(r, "ref"))
	if ref == "" {
		http.Error(w, "container ref is required", http.StatusBadRequest)
		return
	}

	containerService, err := h.hostService.FindContainerByLabel(containerLinkLabel, ref, h.resolveLabels(r))
	if err != nil {
		status := http.StatusNotFound
		if strings.Contains(strings.ToLower(err.Error()), "multiple containers matched") {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	base := strings.TrimSuffix(h.config.Base, "/")
	redirectTarget := base + "/container/" + url.PathEscape(containerService.Container.ID) + suffix
	http.Redirect(w, r, redirectTarget, http.StatusFound)
}
