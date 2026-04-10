package web

import (
	"net/http"
	"sync"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	sseWriter, err := support_web.NewSSEWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating sse writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	events := make(chan container.ContainerEvent)
	stats := make(chan container.ContainerStat)
	availableHosts := make(chan container.Host)

	h.hostService.SubscribeEventsAndStats(r.Context(), events, stats)
	h.hostService.SubscribeAvailableHosts(r.Context(), availableHosts)

	userLabels := h.config.Labels
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
	}

	type hostContainersResult struct {
		host       string
		containers []container.Container
		err        error
	}

	initialContainers := make(chan hostContainersResult)
	var initialWG sync.WaitGroup
	hosts := h.hostService.Hosts()

	for _, host := range hosts {
		initialWG.Add(1)
		go func(host container.Host) {
			defer initialWG.Done()
			containers, err := h.hostService.ListContainersForHost(host.ID, userLabels)
			select {
			case initialContainers <- hostContainersResult{host: host.ID, containers: containers, err: err}:
			case <-r.Context().Done():
			}
		}(host)
	}

	go func() {
		initialWG.Wait()
		close(initialContainers)
	}()

	if err := sseWriter.Event("containers-changed", []container.Container{}); err != nil {
		log.Error().Err(err).Msg("error writing initial empty containers to event stream")
		return
	}

	go sendBeaconEvent(h, r, 0)

	for {
		select {
		case result, ok := <-initialContainers:
			if !ok {
				initialContainers = nil
				continue
			}

			if result.err != nil {
				log.Warn().Err(result.err).Str("host", result.host).Msg("error listing containers for host during initial sync")
				if hostNotAvailableError, ok := result.err.(*docker_support.HostUnavailableError); ok {
					if err := sseWriter.Event("update-host", hostNotAvailableError.Host); err != nil {
						log.Error().Err(err).Msg("error writing event to event stream")
						return
					}
				}
				continue
			}

			if err := sseWriter.Event("containers-changed", result.containers); err != nil {
				log.Error().Err(err).Str("host", result.host).Msg("error writing containers to event stream")
				return
			}
		case host := <-availableHosts:
			if err := sseWriter.Event("update-host", host); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
				return
			}
		case stat := <-stats:
			if err := sseWriter.Event("container-stat", stat); err != nil {
				log.Error().Err(err).Msg("error writing event to event stream")
				return
			}
		case event, ok := <-events:
			if !ok {
				return
			}
			log.Trace().Str("event", event.Name).Str("id", event.ActorID).Msg("container event from store")
			switch event.Name {
			case "start", "die", "destroy", "rename":
				if event.Name == "start" || event.Name == "rename" {
					if containers, err := h.hostService.ListContainersForHost(event.Host, userLabels); err == nil {
						log.Debug().Str("host", event.Host).Int("count", len(containers)).Msg("updating containers for host")
						if err := sseWriter.Event("containers-changed", containers); err != nil {
							log.Error().Err(err).Msg("error writing containers to event stream")
							return
						}
					}
				}

				if err := sseWriter.Event("container-event", event); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}

			case "update":
				if err := sseWriter.Event("container-updated", event.Container); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}
			case "health_status: healthy", "health_status: unhealthy":
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}
				payload := map[string]string{
					"actorId": event.ActorID,
					"health":  healthy,
				}

				if err := sseWriter.Event("container-health", payload); err != nil {
					log.Error().Err(err).Msg("error writing event to event stream")
					return
				}
			}
		case <-r.Context().Done():
			return
		}
	}
}

func sendBeaconEvent(h *handler, r *http.Request, runningContainers int) {
	if h.config.NoAnalytics {
		return
	}
	b := types.BeaconEvent{
		AuthProvider:      string(h.config.Authorization.Provider),
		Browser:           r.Header.Get("User-Agent"),
		Clients:           len(h.hostService.Hosts()),
		HasActions:        h.config.EnableActions,
		HasCustomAddress:  h.config.Addr != ":8080",
		HasCustomBase:     h.config.Base != "/",
		HasHostname:       h.config.Hostname != "",
		Name:              "events",
		RunningContainers: runningContainers,
		Version:           h.config.Version,
	}

	local, err := h.hostService.LocalHost()
	if err == nil {
		b.ServerID = local.ID
	}

	if err := analytics.SendBeacon(b); err != nil {
		log.Debug().Err(err).Msg("error sending beacon")
	}
}
