package web

import (
	"context"
	"strings"

	"github.com/amir20/dozzle/internal/cardreports"
	"github.com/amir20/dozzle/internal/container"
)

func (h *handler) startContainerReportCacheInvalidation() {
	if h.cardReportManager == nil {
		return
	}

	containers := make(chan container.Container, 32)
	h.hostService.SubscribeContainersStarted(context.Background(), containers, func(*container.Container) bool { return true })

	go func() {
		for c := range containers {
			h.invalidateContainerReportCache(c)
		}
	}()
}

func (h *handler) invalidateContainerReportCache(c container.Container) {
	stablePrefix := "event:" + detailReportStableContainerKey(c.Host, c) + ":"
	shouldInvalidate := false
	for _, report := range h.cardReportManager.Reports() {
		if report.Enabled && report.RefreshMode == cardreports.RefreshModeContainerUpdate {
			shouldInvalidate = true
			break
		}
	}
	if !shouldInvalidate {
		return
	}

	h.reportCache.Range(func(key, _ any) bool {
		cacheKey, ok := key.(string)
		if ok && strings.HasPrefix(cacheKey, stablePrefix) {
			h.reportCache.Delete(key)
		}
		return true
	})
}
