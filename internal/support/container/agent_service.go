package container_support

import (
	"context"
	"io"
	"runtime"
	"strings"
	"sync/atomic"

	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/trivy"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

type agentService struct {
	client *agent.Client
	host   atomic.Pointer[container.Host]
}

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
	log.Debug().Interface("labels", labels).Str("caller", caller).Msg("agent service list containers started")
	containers, err := a.client.ListContainers(ctx, labels)
	if err != nil {
		log.Warn().Err(err).Interface("labels", labels).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service list containers failed")
		return nil, err
	}
	log.Debug().Interface("labels", labels).Int("count", len(containers)).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service list containers completed")
	return containers, nil
}

func (a *agentService) Host(ctx context.Context) (container.Host, error) {
	started := time.Now()
	caller := agentDiagnosticCaller(3)
	host, err := a.client.Host(ctx)
	if err != nil {
		log.Warn().Err(err).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service host failed")
		if cached := a.host.Load(); cached != nil {
			h := *cached
			h.Available = false
			return h, err
		}
		return container.Host{Available: false}, err
	}

	a.host.Store(&host)
	log.Debug().Str("host", host.Name).Str("caller", caller).Dur("elapsed", time.Since(started)).Msg("agent service host completed")
	return host, nil
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
