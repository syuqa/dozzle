package web

import (
	"context"
	"io/fs"
	"time"

	"net/http"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/amir20/dozzle/internal/trivy"
	"github.com/amir20/dozzle/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

type ReleaseCheckMode string

const (
	Automatic ReleaseCheckMode = "automatic"
	Manual    ReleaseCheckMode = "manual"
)

type AuthProvider string

const (
	NONE          AuthProvider = "none"
	SIMPLE        AuthProvider = "simple"
	FORWARD_PROXY AuthProvider = "forward-proxy"
)

// Config is a struct for configuring the web service
type Config struct {
	Base                string
	Addr                string
	Version             string
	Hostname            string
	AppName             string
	AppLogoURL          string
	ScanStatusEndpoint  string
	LogIncidentEndpoint string
	LogIncidentDebug    bool
	NoAnalytics         bool
	Dev                 bool
	Mode                string
	Authorization       Authorization
	EnableActions       bool
	EnableContainerScan bool
	TrivyPath           string
	ScanManager         ScanManager
	EnableShell         bool
	DisableAvatars      bool
	ReleaseCheckMode    ReleaseCheckMode
	Labels              container.ContainerLabels
}

type Authorization struct {
	Provider   AuthProvider
	Authorizer Authorizer
	TTL        time.Duration
	LogoutUrl  string
}

type Authorizer interface {
	AuthMiddleware(http.Handler) http.Handler
	CreateToken(string, string) (string, error)
}

type HostService interface {
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
	ListContainersForHost(host string, labels container.ContainerLabels) ([]container.Container, error)
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	ListAllContainersFiltered(userFilter container.ContainerLabels, filter container_support.ContainerFilter) ([]container.Container, []error)
	SubscribeEventsAndStats(ctx context.Context, events chan<- container.ContainerEvent, stats chan<- container.ContainerStat)
	SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter)
	Hosts() []container.Host
	LocalHost() (container.Host, error)
	SubscribeAvailableHosts(ctx context.Context, hosts chan<- container.Host)
	LocalClients() []container.Client
	LocalClientServices() []container_support.ClientService
	// Notification methods
	AddSubscription(sub *notification.Subscription) error
	RemoveSubscription(id int)
	ReplaceSubscription(sub *notification.Subscription) error
	UpdateSubscription(id int, updates map[string]any) error
	Subscriptions() []*notification.Subscription
	AddDispatcher(d dispatcher.Dispatcher) int
	UpdateDispatcher(id int, d dispatcher.Dispatcher)
	RemoveDispatcher(id int)
	Dispatchers() []notification.DispatcherConfig
	FetchAgentNotificationStats() map[int]types.SubscriptionStats
}

type handler struct {
	content      fs.FS
	config       *Config
	hostService  HostService
	trivyScanner TrivyScanner
	scanManager  ScanManager
}

func CreateServer(hostService HostService, content fs.FS, config Config) *http.Server {
	handler := &handler{
		content:     content,
		config:      &config,
		hostService: hostService,
		scanManager: config.ScanManager,
	}
	if config.EnableContainerScan {
		handler.trivyScanner = trivy.NewScanner(config.TrivyPath)
	}

	return &http.Server{Addr: config.Addr, Handler: createRouter(handler)}
}

var fileServer http.Handler

func createRouter(h *handler) *chi.Mux {
	fileServer = http.FileServer(http.FS(h.content))
	base := h.config.Base
	r := chi.NewRouter()

	if !h.config.Dev {
		r.Use(h.cspHeaders)
	}

	if h.config.Authorization.Provider != NONE && h.config.Authorization.Authorizer == nil {
		log.Fatal().Msg("Authorization provider is set but no authorizer is provided")
	}

	r.Route(base, func(r chi.Router) {
		if h.config.Authorization.Provider != NONE {
			r.Use(h.config.Authorization.Authorizer.AuthMiddleware)
		}

		r.Route("/api", func(r chi.Router) {
			// Authenticated routes
			r.Group(func(r chi.Router) {
				if h.config.Authorization.Provider != NONE {
					r.Use(auth.RequireAuthentication)
				}

				// Log streams
				r.Get("/hosts/{host}/containers/{id}/logs/stream", h.streamContainerLogs)
				r.Get("/hosts/{host}/logs/stream", h.streamHostLogs)
				r.Get("/hosts/{host}/containers/{id}/logs", h.fetchLogsBetweenDates)
				r.Post("/hosts/{host}/containers/{id}/logs/match-incident", h.matchLogIncident)
				r.Get("/hosts/{host}/logs/mergedStream/{ids}", h.streamLogsMerged)
				r.Get("/containers/{hostIds}/download", h.downloadLogs) // formatted as host:container,host:container
				r.Get("/labels/{labels}/logs/stream", h.streamLogsWithLabels)
				r.Get("/groups/{group}/logs/stream", h.streamGroupedLogs)
				r.Get("/events/stream", h.streamEvents)

				// Action
				if h.config.EnableActions {
					r.Post("/hosts/{host}/containers/{id}/actions/{action}", h.containerActions)
				}
				if h.config.EnableContainerScan {
					r.Post("/hosts/{host}/containers/{id}/scan", h.scanContainer)
					r.Get("/hosts/{host}/containers/{id}/scan", h.getContainerScan)
					r.Post("/hosts/{host}/containers/{id}/scan/run", h.runContainerScan)
					r.Patch("/hosts/{host}/containers/{id}/scan/schedule", h.updateContainerScanSchedule)
					r.Get("/hosts/{host}/containers/{id}/scan/status", h.getContainerScanStatus)
					r.Get("/scans/summary", h.getScanSummary)
					r.Get("/scans/alerts", h.listScanAlerts)
					r.Post("/scans/alerts", h.createScanAlert)
					r.Put("/scans/alerts/{id}", h.updateScanAlert)
					r.Delete("/scans/alerts/{id}", h.deleteScanAlert)
				}
				if h.config.EnableShell {
					r.Get("/hosts/{host}/containers/{id}/attach", h.attach)
					r.Get("/hosts/{host}/containers/{id}/exec", h.exec)
				}

				if !h.config.DisableAvatars {
					r.Get("/profile/avatar", h.avatar)
				}
				r.Patch("/profile", h.updateProfile)
				r.Get("/version", h.version)
				if log.Debug().Enabled() {
					r.Get("/debug/store", h.debugStore)
				}

				// Notifications API
				r.Route("/notifications", func(r chi.Router) {
					r.Get("/rules", h.listNotificationRules)
					r.Post("/rules", h.createNotificationRule)
					r.Get("/rules/{id}", h.getNotificationRule)
					r.Put("/rules/{id}", h.replaceNotificationRule)
					r.Patch("/rules/{id}", h.updateNotificationRule)
					r.Delete("/rules/{id}", h.deleteNotificationRule)

					r.Get("/dispatchers", h.listDispatchers)
					r.Post("/dispatchers", h.createDispatcher)
					r.Get("/dispatchers/{id}", h.getDispatcher)
					r.Put("/dispatchers/{id}", h.updateDispatcher)
					r.Delete("/dispatchers/{id}", h.deleteDispatcher)

					r.Post("/preview", h.previewExpression)
					r.Post("/test-webhook", h.testWebhook)
				})

				// Releases API
				r.Get("/releases", h.getReleases)

				// Cloud API
				r.Get("/cloud/status", h.cloudStatus)
			})

			// Public API routes
			if h.config.Authorization.Provider == SIMPLE {
				r.Post("/token", h.createToken)
				r.Delete("/token", h.deleteToken)
			}

			// Cloud callback (public, handles OAuth-style code exchange)
			r.Get("/cloud/callback", h.cloudCallback)
		})

		r.Get("/healthcheck", h.healthcheck)

		defaultHandler := http.StripPrefix(strings.Replace(base+"/", "//", "/", 1), http.HandlerFunc(h.index))
		r.With(Brotli).Get("/*", func(w http.ResponseWriter, req *http.Request) {
			defaultHandler.ServeHTTP(w, req)
		})
	})

	if base != "/" {
		r.Get(base, func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		})
	}

	if log.Debug().Enabled() {
		r.Mount("/debug", middleware.Profiler())
	}

	return r
}

func hostKey(r *http.Request) string {
	host := chi.URLParam(r, "host")

	if host == "" {
		log.Fatal().Str("url", r.URL.String()).Msg("Host parameter not found in the URL path")
	}

	return host
}
