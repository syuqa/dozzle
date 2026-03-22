package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/amir20/dozzle/internal/scan"
	"github.com/go-chi/chi/v5"
)

type ScanManager interface {
	GetState(host, id string) *scan.ContainerScanState
	RunScan(ctx context.Context, host, id string, force bool) (*scan.ContainerScanState, error)
	SetSchedule(host, id string, schedule scan.ScanSchedule) (*scan.ContainerScanState, error)
	Summary() scan.DashboardSummary
	Alerts() []*scan.ScanAlert
	AddAlert(alert *scan.ScanAlert) (*scan.ScanAlert, error)
	UpdateAlert(id int, next *scan.ScanAlert) (*scan.ScanAlert, error)
	DeleteAlert(id int)
}

func (h *handler) getScanSummary(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	writeJSON(w, http.StatusOK, h.scanManager.Summary())
}

func (h *handler) getContainerScan(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	state := h.scanManager.GetState(hostKey(r), chi.URLParam(r, "id"))
	if state == nil {
		writeError(w, http.StatusNotFound, "scan state not found")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *handler) runContainerScan(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	force := r.URL.Query().Get("force") == "1" || r.URL.Query().Get("force") == "true"
	state, err := h.scanManager.RunScan(r.Context(), hostKey(r), chi.URLParam(r, "id"), force)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *handler) updateContainerScanSchedule(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	var input scan.ScanSchedule
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.scanManager.SetSchedule(hostKey(r), chi.URLParam(r, "id"), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *handler) listScanAlerts(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	writeJSON(w, http.StatusOK, h.scanManager.Alerts())
}

func (h *handler) createScanAlert(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	var input scan.ScanAlert
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	alert, err := h.scanManager.AddAlert(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, alert)
}

func (h *handler) updateScanAlert(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var input scan.ScanAlert
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	alert, err := h.scanManager.UpdateAlert(id, &input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, alert)
}

func (h *handler) deleteScanAlert(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		writeError(w, http.StatusServiceUnavailable, "scan manager is not configured")
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	h.scanManager.DeleteAlert(id)
	w.WriteHeader(http.StatusNoContent)
}
