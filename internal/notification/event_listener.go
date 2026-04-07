package notification

import (
	"context"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

// ContainerEventEnvelope pairs an event with resolved container and host metadata.
type ContainerEventEnvelope struct {
	Event     container.ContainerEvent
	Container container.Container
	Host      container.Host
}

// ContainerEventListener subscribes to container events and enriches them with metadata.
type ContainerEventListener struct {
	clients   []container_support.ClientService
	channel   chan *ContainerEventEnvelope
	parentCtx context.Context
	cache     *TTLCache[string, containerInfo]
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
	rawEvents := make(chan container.ContainerEvent, 1000)
	for _, client := range l.clients {
		client.SubscribeEvents(l.parentCtx, rawEvents)
	}

	go l.enrich(l.parentCtx, rawEvents)
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
			c, host, err := l.resolveEvent(ctx, event)
			if err != nil {
				log.Debug().Err(err).Str("containerID", event.ActorID).Str("event", event.Name).Msg("Failed to resolve container event")
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
		// Keep cached metadata available for stop/destroy style events that can race with removal.
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
