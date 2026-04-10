package web

import (
	"encoding/json"
	"net/http"

	"github.com/amir20/dozzle/internal/cardreports"
	"github.com/amir20/dozzle/internal/cardtemplates"
)

type CardTemplateManager interface {
	Templates() []cardtemplates.Template
	ReplaceTemplates([]cardtemplates.Template) ([]cardtemplates.Template, error)
}

type CardReportManager interface {
	Reports() []cardreports.Definition
	ReplaceReports([]cardreports.Definition) ([]cardreports.Definition, error)
	Get(string) (cardreports.Definition, bool)
}

func (h *handler) listCardTemplates(w http.ResponseWriter, r *http.Request) {
	if h.cardTemplateManager == nil {
		writeError(w, http.StatusServiceUnavailable, "card template manager is not configured")
		return
	}

	writeJSON(w, http.StatusOK, h.cardTemplateManager.Templates())
}

func (h *handler) replaceCardTemplates(w http.ResponseWriter, r *http.Request) {
	if h.cardTemplateManager == nil {
		writeError(w, http.StatusServiceUnavailable, "card template manager is not configured")
		return
	}

	var input []cardtemplates.Template
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	templates, err := h.cardTemplateManager.ReplaceTemplates(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, templates)
}

func (h *handler) listCardReports(w http.ResponseWriter, r *http.Request) {
	if h.cardReportManager == nil {
		writeError(w, http.StatusServiceUnavailable, "card report manager is not configured")
		return
	}
	writeJSON(w, http.StatusOK, h.cardReportManager.Reports())
}

func (h *handler) replaceCardReports(w http.ResponseWriter, r *http.Request) {
	if h.cardReportManager == nil {
		writeError(w, http.StatusServiceUnavailable, "card report manager is not configured")
		return
	}

	var input []cardreports.Definition
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	reports, err := h.cardReportManager.ReplaceReports(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, reports)
}
