package notification

import (
	"context"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

var allowedEventNames = map[string]bool{
	"start":         true,
	"stop":          true,
	"die":           true,
	"restart":       true,
	"health_status": true,
	"oom":           true,
}

// ContainerEventEnvelope pairs an event with resolved container and host metadata.
type ContainerEventEnvelope struct {
	Event     container.ContainerEvent
	Container container.Container
	Host      container.Host
}

type ContainerEventListener struct {
	clients    []container_support.ClientService
	channel    chan *ContainerEventEnvelope
	parentCtx  context.Context
	cache      *TTLCache[string, containerInfo]
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

func NewContainerEventListener(ctx context.Context, clients []container_support.ClientService) *ContainerEventListener {
	return &ContainerEventListener{
		clients:   clients,
		channel:   make(chan *ContainerEventEnvelope, 1000),
		parentCtx: ctx,
		cache:     NewTTLCache[string, containerInfo](ctx, 30*time.Second),
	}
}

func (l *ContainerEventListener) Start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancelFunc != nil {
		return
	}

	ctx, cancel := context.WithCancel(l.parentCtx)
	l.cancelFunc = cancel

	rawEvents := make(chan container.ContainerEvent, 1000)
	for _, client := range l.clients {
		client.SubscribeEvents(ctx, rawEvents)
	}

	go l.enrich(ctx, rawEvents)
}

func (l *ContainerEventListener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancelFunc == nil {
		return
	}
	l.cancelFunc()
	l.cancelFunc = nil
}

func (l *ContainerEventListener) enrich(ctx context.Context, rawEvents <-chan container.ContainerEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-rawEvents:
			if !ok {
				return
			}
			if !allowedEventNames[event.Name] {
				continue
			}
			c, host, err := l.resolveEvent(ctx, event)
			if err != nil {
				log.Debug().Err(err).Str("containerID", event.ActorID).Str("event", event.Name).Msg("Failed to resolve container event")
				continue
			}
			if isDozzleContainer(c) {
				continue
			}

			select {
			case l.channel <- &ContainerEventEnvelope{Event: event, Container: c, Host: host}:
			case <-ctx.Done():
				return
			default:
				log.Warn().Str("containerID", event.ActorID).Str("event", event.Name).Msg("Container event channel full, dropping event")
			}
		}
	}
}

func (l *ContainerEventListener) resolveEvent(ctx context.Context, event container.ContainerEvent) (container.Container, container.Host, error) {
	cacheKey := event.Host + ":" + event.ActorID

	if event.Container != nil {
		host := container.Host{ID: event.Host, Name: event.Host}
		for _, client := range l.clients {
			resolvedHost, err := client.Host(ctx)
			if err == nil && resolvedHost.ID == event.Host {
				host = resolvedHost
				break
			}
		}
		c := *event.Container
		if c.Host == "" {
			c.Host = event.Host
		}
		l.cache.Store(cacheKey, containerInfo{container: c, host: host})
		return c, host, nil
	}

	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, client := range l.clients {
		c, err := client.FindContainer(resolveCtx, event.ActorID, nil)
		if err != nil {
			continue
		}
		host, err := client.Host(resolveCtx)
		if err != nil {
			continue
		}
		l.cache.Store(cacheKey, containerInfo{container: c, host: host})
		return c, host, nil
	}

	if cached, ok := l.cache.Load(cacheKey); ok {
		return cached.container, cached.host, nil
	}

	return container.Container{}, container.Host{}, container.ErrContainerNotFound
}

func (l *ContainerEventListener) Channel() <-chan *ContainerEventEnvelope {
	return l.channel
}

func (l *ContainerEventListener) FindContainerWithHost(ctx context.Context, containerID string) (container.Container, container.Host, error) {
	for _, client := range l.clients {
		c, err := client.FindContainer(ctx, containerID, nil)
		if err != nil {
			continue
		}
		host, err := client.Host(ctx)
		if err != nil {
			continue
		}
		return c, host, nil
	}

	return container.Container{}, container.Host{}, container.ErrContainerNotFound
}
