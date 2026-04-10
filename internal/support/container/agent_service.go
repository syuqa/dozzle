package container_support

import (
	"context"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/trivy"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
)

type agentService struct {
	client           *agent.Client
	host             atomic.Pointer[container.Host]
	hostCachedAt     atomic.Int64
	hostSingleflight singleflight.Group
	listCache        sync.Map
	listSingleflight singleflight.Group
}

type cachedContainerList struct {
	Containers []container.Container
	CachedAt   time.Time
}

const (
	agentHostCacheTTL          = 5 * time.Second
	agentListContainersCacheTTL = 3 * time.Second
)

func NewAgentService(client *agent.Client) ClientService {
	return &agentService{
		client: client,
	}
}

func (a *agentService) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	started := time.Now()
	caller := agentDiagnosticCaller(3)
	result, err := a.client.FindContainer(ctx, id, labels)
	if err != nil {
		log.Warn().Err(err).Str("container", id).Interface("labels", labels).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service find container failed")
		return result, err
	}
	log.Debug().Str("container", id).Interface("labels", labels).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service find container completed")
	return result, nil
}

func (a *agentService) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return a.client.StreamRawBytes(ctx, container.ID, from, to, stdTypes)
}

func (a *agentService) LogsBetweenDates(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	log.Debug().Str("container", container.ID).Time("from", from).Time("to", to).Str("caller", agentDiagnosticCaller(3)).Msg("agent service logs between dates started")
	return a.client.LogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (a *agentService) StreamLogs(ctx context.Context, container container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	started := time.Now()
	caller := agentDiagnosticCaller(3)
	log.Debug().Str("container", container.ID).Time("from", from).Str("caller", caller).Msg("agent service stream logs started")
	err := a.client.StreamContainerLogs(ctx, container.ID, from, stdTypes, events)
	if err != nil {
		log.Warn().Err(err).Str("container", container.ID).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service stream logs failed")
		return err
	}
	log.Debug().Str("container", container.ID).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service stream logs completed")
	return nil
}

func (a *agentService) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	started := time.Now()
	caller := agentDiagnosticCaller(3)
	cacheKey := agentLabelsCacheKey(labels)
	log.Debug().Interface("labels", labels).Str("caller", caller).Msg("agent service list containers started")

	if cached, ok := a.cachedList(cacheKey); ok {
		log.Debug().Interface("labels", labels).Str("caller", caller).Int("count", len(cached)).Dur("elapsed", time.Since(started)).Msg("agent service list containers cache hit")
		return cached, nil
	}

	result, err, shared := a.listSingleflight.Do(cacheKey, func() (interface{}, error) {
		if cached, ok := a.cachedList(cacheKey); ok {
			return cached, nil
		}

		containers, err := a.client.ListContainers(ctx, labels)
		if err != nil {
			return nil, err
		}

		a.listCache.Store(cacheKey, cachedContainerList{
			Containers: cloneContainers(containers),
			CachedAt:   time.Now(),
		})
		return cloneContainers(containers), nil
	})
	if err != nil {
		log.Warn().Err(err).Interface("labels", labels).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service list containers failed")
		return nil, err
	}

	containers, _ := result.([]container.Container)
	event := "agent service list containers completed"
	if shared {
		event = "agent service list containers completed via shared result"
	}
	log.Debug().Interface("labels", labels).Int("count", len(containers)).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg(event)
	return containers, nil
}

func (a *agentService) Host(ctx context.Context) (container.Host, error) {
	started := time.Now()
	caller := agentDiagnosticCaller(3)
	if cached, ok := a.cachedHost(); ok {
		log.Debug().Str("host", cached.Name).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service host cache hit")
		return cached, nil
	}

	result, err, shared := a.hostSingleflight.Do("host", func() (interface{}, error) {
		if cached, ok := a.cachedHost(); ok {
			return cached, nil
		}

		host, err := a.client.Host(ctx)
		if err != nil {
			return container.Host{}, err
		}

		a.host.Store(&host)
		a.hostCachedAt.Store(time.Now().UnixNano())
		return host, nil
	})
	if err != nil {
		log.Warn().Err(err).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service host failed")
		if cached := a.host.Load(); cached != nil {
			h := *cached
			h.Available = false
			return h, err
		}
		return container.Host{Available: false}, err
	}

	host, _ := result.(container.Host)
	event := "agent service host completed"
	if shared {
		event = "agent service host completed via shared result"
	}
	log.Debug().Str("host", host.Name).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg(event)
	return host, nil
}

func (a *agentService) cachedHost() (container.Host, bool) {
	cached := a.host.Load()
	if cached == nil {
		return container.Host{}, false
	}

	cachedAt := time.Unix(0, a.hostCachedAt.Load())
	if cachedAt.IsZero() || time.Since(cachedAt) > agentHostCacheTTL {
		return container.Host{}, false
	}

	return *cached, true
}

func (a *agentService) cachedList(cacheKey string) ([]container.Container, bool) {
	value, ok := a.listCache.Load(cacheKey)
	if !ok {
		return nil, false
	}

	entry, ok := value.(cachedContainerList)
	if !ok {
		a.listCache.Delete(cacheKey)
		return nil, false
	}

	if time.Since(entry.CachedAt) > agentListContainersCacheTTL {
		a.listCache.Delete(cacheKey)
		return nil, false
	}

	return cloneContainers(entry.Containers), true
}

func agentLabelsCacheKey(labels container.ContainerLabels) string {
	if !labels.Exists() {
		return "*"
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		values := append([]string(nil), labels[key]...)
		sort.Strings(values)
		parts = append(parts, key+"="+strings.Join(values, ","))
	}

	return strings.Join(parts, "|")
}

func cloneContainers(containers []container.Container) []container.Container {
	if len(containers) == 0 {
		return []container.Container{}
	}

	cloned := make([]container.Container, len(containers))
	copy(cloned, containers)
	return cloned
}

func agentDiagnosticCaller(skip int) string {
	pcs := make([]uintptr, 12)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return "unknown"
	}

	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		name := frame.Function
		if name != "" &&
			!strings.Contains(name, "runtime.") &&
			!strings.Contains(name, "testing.") &&
			!strings.Contains(name, "stretchr/testify") {
			if idx := strings.LastIndex(name, "/"); idx >= 0 {
				name = name[idx+1:]
			}
			return name
		}
		if !more {
			break
		}
	}
	return "unknown"
}

func (a *agentService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	go a.client.StreamStats(ctx, stats)
}

func (a *agentService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	go a.client.StreamEvents(ctx, events)
}

func (d *agentService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	go d.client.StreamNewContainers(ctx, containers)
}

func (a *agentService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return a.client.ContainerAction(ctx, container.ID, action)
}

func (a *agentService) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	return a.client.UpdateContainer(ctx, c.ID, progressCh)
}

func (a *agentService) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
	panic("not implemented")
}

func (a *agentService) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	return a.client.Exec(ctx, c.ID, cmd, events, stdout)
}

func (a *agentService) RunContainerScan(ctx context.Context, id string) (*trivy.Result, error) {
	return a.client.RunContainerScan(ctx, id)
}

func (a *agentService) UpdateNotificationConfig(ctx context.Context, subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	return a.client.UpdateNotificationConfig(ctx, subscriptions, dispatchers)
}

func (a *agentService) GetNotificationStats(ctx context.Context) ([]types.SubscriptionStats, error) {
	return a.client.GetNotificationStats(ctx)
}
