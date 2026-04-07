package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
)

const (
	StateTriggerImageUpdated = "image_updated"
	StateTriggerStopped      = "stopped"
	StateTriggerStarted      = "started"
	StateTriggerError        = "error"
	StateTriggerUnhealthy    = "unhealthy"
	StateTriggerRestarted    = "restarted"
	StateTriggerOOMKilled    = "oom_killed"
)

type pendingStateAlert struct {
	SubscriptionID int
	ContainerID    string
	ContainerKey   string
	Trigger        string
	Version        int64
	ExpectedState  string
	ExpectedHealth string
	ExpectedImage  string
	PreviousState  string
	CurrentState   string
	PreviousImage  string
	CurrentImage   string
	EventName      string
	ExitCode       string
	Attributes     map[string]string
}

func containerStateKey(host, containerID string) string {
	return host + ":" + containerID
}

func (m *Manager) processContainerEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case event, ok := <-m.eventListener.Channel():
			if !ok || event == nil {
				return
			}
			m.processContainerEvent(event)
		}
	}
}

func (m *Manager) processContainerEvent(envelope *ContainerEventEnvelope) {
	current := envelope.Container
	host := envelope.Host

	if isDozzleContainer(current) {
		return
	}

	containerKey := containerStateKey(host.ID, current.ID)
	previous, hadPrevious := m.containerSnapshots.Load(containerKey)
	m.containerSnapshots.Store(containerKey, current)
	if !hadPrevious {
		return
	}

	notificationContainer := FromContainerModel(current, host)
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		if !sub.Enabled || !sub.IsStateAlert() {
			return true
		}
		if !sub.MatchesContainer(notificationContainer) {
			return true
		}

		triggered := detectStateTriggers(previous, current, envelope.Event)
		for _, trigger := range sub.StateTriggers {
			if payload, ok := triggered[trigger]; ok {
				m.queueStateAlert(sub, current, host, payload)
			}
		}

		return true
	})
}

func detectStateTriggers(previous container.Container, current container.Container, event container.ContainerEvent) map[string]pendingStateAlert {
	triggered := make(map[string]pendingStateAlert)

	add := func(trigger string, payload pendingStateAlert) {
		payload.Trigger = trigger
		payload.ContainerID = current.ID
		payload.ContainerKey = containerStateKey(current.Host, current.ID)
		payload.ExpectedState = current.State
		payload.ExpectedHealth = current.Health
		payload.ExpectedImage = current.Image
		payload.PreviousState = previous.State
		payload.CurrentState = current.State
		payload.PreviousImage = previous.Image
		payload.CurrentImage = current.Image
		payload.EventName = event.Name
		payload.Attributes = event.ActorAttributes
		triggered[trigger] = payload
	}

	if previous.Image != "" && current.Image != "" && previous.Image != current.Image {
		add(StateTriggerImageUpdated, pendingStateAlert{})
	}

	started := previous.State != "running" && current.State == "running"
	if event.Name == "start" {
		started = true
	}
	if started {
		add(StateTriggerStarted, pendingStateAlert{})
	}

	stopped := previous.State == "running" && current.State != "running"
	if event.Name == "die" {
		stopped = true
	}
	if stopped {
		add(StateTriggerStopped, pendingStateAlert{ExpectedState: current.State})
	}

	if previous.Health != "unhealthy" && current.Health == "unhealthy" {
		add(StateTriggerUnhealthy, pendingStateAlert{ExpectedHealth: "unhealthy"})
		add(StateTriggerError, pendingStateAlert{ExpectedHealth: "unhealthy"})
	}

	if event.Name == "oom" {
		add(StateTriggerOOMKilled, pendingStateAlert{})
		add(StateTriggerError, pendingStateAlert{})
	}

	if event.Name == "restart" || (started && !previous.FinishedAt.IsZero()) {
		add(StateTriggerRestarted, pendingStateAlert{ExpectedState: "running"})
	}

	exitCode := strings.TrimSpace(event.ActorAttributes["exitCode"])
	if event.Name == "die" && exitCode != "" && exitCode != "0" {
		add(StateTriggerError, pendingStateAlert{ExitCode: exitCode, ExpectedState: current.State})
	}

	return triggered
}

func (m *Manager) queueStateAlert(sub *Subscription, current container.Container, host container.Host, payload pendingStateAlert) {
	payload.SubscriptionID = sub.ID
	payload.ContainerID = current.ID
	payload.ContainerKey = containerStateKey(host.ID, current.ID)

	holdoff := sub.GetHoldoffSeconds()
	if holdoff <= 0 {
		m.fireStateAlert(sub, current, host, payload)
		return
	}

	pendingKey := fmt.Sprintf("%d:%s:%s", sub.ID, payload.ContainerKey, payload.Trigger)
	version := m.stateCheckCounter.Add(1)
	payload.Version = version
	m.pendingStateChecks.Store(pendingKey, version)

	go func() {
		timer := time.NewTimer(time.Duration(holdoff) * time.Second)
		defer timer.Stop()
		defer m.pendingStateChecks.Delete(pendingKey)

		select {
		case <-m.ctx.Done():
			return
		case <-timer.C:
		}

		if currentVersion, ok := m.pendingStateChecks.Load(pendingKey); !ok || currentVersion != version {
			return
		}

		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		defer cancel()

		resolved, resolvedHost, err := m.eventListener.FindContainerWithHost(ctx, payload.ContainerID)
		if err != nil {
			return
		}

		notificationContainer := FromContainerModel(resolved, resolvedHost)
		if !sub.Enabled || !sub.MatchesContainer(notificationContainer) {
			return
		}
		if !stateTriggerStillValid(payload, resolved) {
			return
		}

		m.fireStateAlert(sub, resolved, resolvedHost, payload)
	}()
}

func stateTriggerStillValid(payload pendingStateAlert, current container.Container) bool {
	switch payload.Trigger {
	case StateTriggerStarted, StateTriggerRestarted:
		return current.State == "running"
	case StateTriggerStopped:
		return current.State != "running"
	case StateTriggerUnhealthy:
		return current.Health == "unhealthy"
	case StateTriggerImageUpdated:
		return payload.ExpectedImage != "" && current.Image == payload.ExpectedImage
	case StateTriggerOOMKilled:
		return current.State != "running"
	case StateTriggerError:
		if payload.ExitCode != "" {
			return current.State != "running"
		}
		if payload.ExpectedHealth == "unhealthy" {
			return current.Health == "unhealthy"
		}
		return current.State != "running"
	default:
		return true
	}
}

func (m *Manager) fireStateAlert(sub *Subscription, current container.Container, host container.Host, payload pendingStateAlert) {
	if sub.IsStateCooldownActive(payload.ContainerKey) {
		return
	}

	sub.SetStateCooldown(payload.ContainerKey)
	sub.AddTriggeredContainer(current.ID)
	sub.TriggerCount.Add(1)
	now := time.Now()
	sub.LastTriggeredAt.Store(&now)

	notificationContainer := FromContainerModel(current, host)
	stateDetail := formatStateDetail(current, payload)
	notification := types.Notification{
		ID:        fmt.Sprintf("%s-state-%d", current.ID, time.Now().UnixNano()),
		Type:      types.StateNotification,
		Detail:    stateDetail,
		Container: notificationContainer,
		State: &types.NotificationState{
			Trigger:       payload.Trigger,
			Event:         payload.EventName,
			PreviousState: payload.PreviousState,
			CurrentState:  current.State,
			PreviousImage: payload.PreviousImage,
			CurrentImage:  current.Image,
			ExitCode:      payload.ExitCode,
			Attributes:    payload.Attributes,
		},
		Subscription: types.SubscriptionConfig{
			ID:                  sub.ID,
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        sub.DispatcherID,
			ContainerExpression: sub.ContainerExpression,
			Cooldown:            sub.Cooldown,
			StateTriggers:       append([]string(nil), sub.StateTriggers...),
			HoldoffSeconds:      sub.HoldoffSeconds,
		},
		Timestamp: now,
	}

	go m.sendSubscriptionNotification(sub, notification)
}

func formatStateDetail(current container.Container, payload pendingStateAlert) string {
	switch payload.Trigger {
	case StateTriggerImageUpdated:
		return fmt.Sprintf("Container %s image changed from %s to %s", current.Name, payload.PreviousImage, current.Image)
	case StateTriggerStarted:
		return fmt.Sprintf("Container %s started", current.Name)
	case StateTriggerStopped:
		return fmt.Sprintf("Container %s stopped", current.Name)
	case StateTriggerRestarted:
		return fmt.Sprintf("Container %s restarted", current.Name)
	case StateTriggerUnhealthy:
		return fmt.Sprintf("Container %s became unhealthy", current.Name)
	case StateTriggerOOMKilled:
		return fmt.Sprintf("Container %s was OOM killed", current.Name)
	case StateTriggerError:
		if payload.ExitCode != "" {
			return fmt.Sprintf("Container %s exited with error code %s", current.Name, payload.ExitCode)
		}
		if current.Health == "unhealthy" {
			return fmt.Sprintf("Container %s reported an unhealthy state", current.Name)
		}
		return fmt.Sprintf("Container %s reported an error", current.Name)
	default:
		return fmt.Sprintf("Container %s changed state", current.Name)
	}
}
