package notification

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
)

// Manager manages notification subscriptions and dispatches notifications
type Manager struct {
	subscriptions       *xsync.Map[int, *Subscription]
	dispatchers         *xsync.Map[int, dispatcher.Dispatcher]
	dispatcherConfigs   *xsync.Map[int, DispatcherConfig]
	templates           *xsync.Map[int, *NotificationTemplate]
	subscriptionCounter atomic.Int32
	dispatcherCounter   atomic.Int32
	templateCounter     atomic.Int32
	listener            *ContainerLogListener
	statsListener       *ContainerStatsListener
	eventListener       *ContainerEventListener
	ctx                 context.Context
	cancel              context.CancelFunc
	sendSem             *semaphore.Weighted
	containerSnapshots  *xsync.Map[string, container.Container]
	identitySnapshots   *xsync.Map[string, container.Container]
	pendingStateChecks  *xsync.Map[string, int64]
	stateCheckCounter   atomic.Int64
}

// NewManager creates a new notification manager
func NewManager(listener *ContainerLogListener, statsListener *ContainerStatsListener, eventListener *ContainerEventListener) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		subscriptions:      xsync.NewMap[int, *Subscription](),
		dispatchers:        xsync.NewMap[int, dispatcher.Dispatcher](),
		dispatcherConfigs:  xsync.NewMap[int, DispatcherConfig](),
		templates:          xsync.NewMap[int, *NotificationTemplate](),
		listener:           listener,
		statsListener:      statsListener,
		eventListener:      eventListener,
		ctx:                ctx,
		cancel:             cancel,
		sendSem:            semaphore.NewWeighted(5),
		containerSnapshots: xsync.NewMap[string, container.Container](),
		identitySnapshots:  xsync.NewMap[string, container.Container](),
		pendingStateChecks: xsync.NewMap[string, int64](),
	}

	// Start processing log events from the listener
	go m.processLogEvents()

	// Start processing stat events from the stats listener
	go m.processStatEvents()

	// Start processing container lifecycle events
	go m.processContainerEvents()

	return m
}

// Start initializes the manager and starts the log listener
func (m *Manager) Start() error {
	for _, c := range m.listener.ListContainers() {
		m.containerSnapshots.Store(containerStateKey(c.Host, c.ID), c)
		m.identitySnapshots.Store(containerIdentityKey(c.Host, c), c)
	}
	m.eventListener.Start()
	return m.listener.Start(m)
}

// ShouldListenToContainer implements ContainerMatcher interface
// Only matches log-based subscriptions (metric-only subscriptions don't need log streaming)
func (m *Manager) ShouldListenToContainer(c container.Container) bool {
	// Pass empty host for matching - host fields aren't used in container expressions
	notificationContainer := FromContainerModel(c, container.Host{})

	shouldListen := false
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		if sub.Enabled && sub.LogExpression != "" && sub.MatchesContainer(notificationContainer) {
			shouldListen = true
			return false
		}
		return true
	})
	return shouldListen
}

// AddSubscription adds a new subscription with compiled expressions
func (m *Manager) AddSubscription(sub *Subscription) error {
	// Auto-increment ID using atomic counter
	sub.ID = int(m.subscriptionCounter.Add(1))
	sub.Enabled = true
	sub.MetricCooldowns = xsync.NewMap[string, time.Time]()
	sub.MetricSampleBuffers = xsync.NewMap[string, *utils.RingBuffer[bool]]()
	sub.StateCooldowns = xsync.NewMap[string, time.Time]()
	sub.EventCooldowns = xsync.NewMap[string, time.Time]()

	if err := sub.CompileExpressions(); err != nil {
		return err
	}
	if err := sub.Validate(); err != nil {
		return err
	}

	m.subscriptions.Store(sub.ID, sub)
	log.Debug().Str("name", sub.Name).Int("id", sub.ID).Msg("Added subscription")

	m.updateListeners()

	return nil
}

// RemoveSubscription removes a subscription by ID
func (m *Manager) RemoveSubscription(id int) {
	if sub, ok := m.subscriptions.LoadAndDelete(id); ok {
		log.Debug().Int("id", id).Str("name", sub.Name).Msg("Removed subscription")

		m.updateListeners()
	}
}

// ReplaceSubscription replaces a subscription with new data
func (m *Manager) ReplaceSubscription(sub *Subscription) error {
	sub.MetricCooldowns = xsync.NewMap[string, time.Time]()
	sub.MetricSampleBuffers = xsync.NewMap[string, *utils.RingBuffer[bool]]()
	sub.StateCooldowns = xsync.NewMap[string, time.Time]()
	sub.EventCooldowns = xsync.NewMap[string, time.Time]()

	if err := sub.CompileExpressions(); err != nil {
		return err
	}
	if err := sub.Validate(); err != nil {
		return err
	}

	// Preserve enabled state from existing subscription if it exists
	if existing, ok := m.subscriptions.Load(sub.ID); ok {
		sub.Enabled = existing.Enabled
	} else {
		sub.Enabled = true
	}

	m.subscriptions.Store(sub.ID, sub)
	log.Debug().Str("name", sub.Name).Int("id", sub.ID).Msg("Replaced subscription")

	m.updateListeners()

	return nil
}

// UpdateSubscription updates a subscription with the provided fields
func (m *Manager) UpdateSubscription(id int, updates map[string]any) error {
	var updateErr error
	_, ok := m.subscriptions.Compute(id, func(sub *Subscription, loaded bool) (*Subscription, xsync.ComputeOp) {
		if !loaded {
			updateErr = fmt.Errorf("subscription not found")
			return nil, xsync.CancelOp
		}

		// Clone the subscription
		updated := &Subscription{
			ID:                    sub.ID,
			Name:                  sub.Name,
			Enabled:               sub.Enabled,
			DispatcherID:          sub.DispatcherID,
			ContainerExpression:   sub.ContainerExpression,
			ContainerProgram:      sub.ContainerProgram,
			LogExpression:         sub.LogExpression,
			LogProgram:            sub.LogProgram,
			MetricExpression:      sub.MetricExpression,
			MetricProgram:         sub.MetricProgram,
			EventExpression:       sub.EventExpression,
			EventProgram:          sub.EventProgram,
			Cooldown:              sub.Cooldown,
			SampleWindow:          sub.SampleWindow,
			StateTriggers:         append([]string(nil), sub.StateTriggers...),
			HoldoffSeconds:        sub.HoldoffSeconds,
			Template:              sub.Template,
			TemplateID:            sub.TemplateID,
			MetricCooldowns:       sub.MetricCooldowns,
			StateCooldowns:        sub.StateCooldowns,
			EventCooldowns:        sub.EventCooldowns,
			MetricSampleBuffers:   sub.MetricSampleBuffers,
			TriggeredContainerIDs: sub.TriggeredContainerIDs,
		}

		// Preserve runtime stats (atomics can't be copied in struct literal)
		updated.TriggerCount.Store(sub.TriggerCount.Load())
		updated.LastTriggeredAt.Store(sub.LastTriggeredAt.Load())

		// Apply updates to the clone
		for key, value := range updates {
			switch key {
			case "name":
				if name, ok := value.(string); ok {
					updated.Name = name
				}
			case "enabled":
				if enabled, ok := value.(bool); ok {
					updated.Enabled = enabled
				}
			case "dispatcherId":
				if dispatcherID, ok := value.(int); ok {
					updated.DispatcherID = dispatcherID
				}
			case "containerExpression":
				if exprStr, ok := value.(string); ok {
					program, err := expr.Compile(exprStr, expr.Env(types.NotificationContainer{}))
					if err != nil {
						updateErr = fmt.Errorf("failed to compile container expression: %w", err)
						return nil, xsync.CancelOp
					}
					updated.ContainerExpression = exprStr
					updated.ContainerProgram = program
				}
			case "logExpression":
				if exprStr, ok := value.(string); ok {
					if exprStr != "" {
						program, err := expr.Compile(exprStr, expr.Env(types.NotificationLog{}))
						if err != nil {
							updateErr = fmt.Errorf("failed to compile log expression: %w", err)
							return nil, xsync.CancelOp
						}
						updated.LogExpression = exprStr
						updated.LogProgram = program
					} else {
						updated.LogExpression = ""
						updated.LogProgram = nil
					}
				}
			case "metricExpression":
				if exprStr, ok := value.(string); ok {
					if exprStr != "" {
						program, err := expr.Compile(exprStr, expr.Env(types.NotificationStat{}))
						if err != nil {
							updateErr = fmt.Errorf("failed to compile metric expression: %w", err)
							return nil, xsync.CancelOp
						}
						updated.MetricExpression = exprStr
						updated.MetricProgram = program
					} else {
						updated.MetricExpression = ""
						updated.MetricProgram = nil
					}
				}
			case "eventExpression":
				if exprStr, ok := value.(string); ok {
					if exprStr != "" {
						program, err := expr.Compile(exprStr, expr.Env(types.NotificationEvent{}))
						if err != nil {
							updateErr = fmt.Errorf("failed to compile event expression: %w", err)
							return nil, xsync.CancelOp
						}
						updated.EventExpression = exprStr
						updated.EventProgram = program
					} else {
						updated.EventExpression = ""
						updated.EventProgram = nil
					}
				}
			case "cooldown":
				if cd, ok := value.(int); ok {
					updated.Cooldown = cd
				}
			case "sampleWindow":
				if sw, ok := value.(int); ok {
					updated.SampleWindow = sw
					updated.MetricSampleBuffers = xsync.NewMap[string, *utils.RingBuffer[bool]]()
				}
			case "stateTriggers":
				if triggers, ok := value.([]string); ok {
					updated.StateTriggers = append([]string(nil), triggers...)
				}
			case "holdoffSeconds":
				if holdoff, ok := value.(int); ok {
					updated.HoldoffSeconds = holdoff
				}
			case "template":
				if templateText, ok := value.(string); ok {
					updated.Template = templateText
				}
			case "templateId":
				if templateID, ok := value.(int); ok {
					updated.TemplateID = templateID
				}
			}
		}

		if err := updated.Validate(); err != nil {
			updateErr = err
			return nil, xsync.CancelOp
		}

		return updated, xsync.UpdateOp
	})

	if updateErr != nil {
		return updateErr
	}

	if !ok {
		return fmt.Errorf("subscription not found")
	}

	log.Debug().Int("id", id).Interface("updates", updates).Msg("Updated subscription")

	m.updateListeners()

	return nil
}

// updateListeners updates log and stats listeners based on current subscriptions
func (m *Manager) updateListeners() {
	m.listener.UpdateStreams()

	hasMetric := false
	hasEvent := false
	hasState := false
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		if sub.Enabled && sub.IsMetricAlert() {
			hasMetric = true
		}
		if sub.Enabled && sub.IsEventAlert() {
			hasEvent = true
		}
		if sub.Enabled && sub.IsStateAlert() {
			hasState = true
		}
		if hasMetric && hasEvent && hasState {
			return false
		}
		return true
	})

	if hasMetric {
		m.statsListener.Start()
	} else {
		m.statsListener.Stop()
	}

	if hasEvent || hasState {
		m.eventListener.Start()
	} else {
		m.eventListener.Stop()
	}
}

// AddDispatcher adds a dispatcher config and returns its auto-generated ID
func (m *Manager) AddDispatcher(config DispatcherConfig) (int, error) {
	id := int(m.dispatcherCounter.Add(1))
	config.ID = id
	d, err := m.createDispatcher(config)
	if err != nil {
		return 0, err
	}
	m.dispatcherConfigs.Store(id, config)
	m.dispatchers.Store(id, d)
	log.Debug().Int("id", id).Msg("Added dispatcher")
	return id, nil
}

// UpdateDispatcher updates a dispatcher by ID
func (m *Manager) UpdateDispatcher(id int, config DispatcherConfig) error {
	config.ID = id
	d, err := m.createDispatcher(config)
	if err != nil {
		return err
	}
	m.dispatcherConfigs.Store(id, config)
	m.dispatchers.Store(id, d)
	log.Debug().Int("id", id).Msg("Updated dispatcher")
	return nil
}

// RemoveDispatcher removes a dispatcher by ID
func (m *Manager) RemoveDispatcher(id int) {
	if _, ok := m.dispatchers.LoadAndDelete(id); ok {
		m.dispatcherConfigs.Delete(id)
		log.Debug().Int("id", id).Msg("Removed dispatcher")
	}
}

// Subscriptions returns all subscriptions sorted by ID
func (m *Manager) Subscriptions() []*Subscription {
	result := make([]*Subscription, 0)
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		result = append(result, sub)
		return true
	})
	slices.SortFunc(result, func(a, b *Subscription) int {
		return a.ID - b.ID
	})
	return result
}

// GetNotificationStats returns runtime stats for all subscriptions
func (m *Manager) GetNotificationStats() []types.SubscriptionStats {
	var stats []types.SubscriptionStats
	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		var containerIDs []string
		if sub.TriggeredContainerIDs != nil {
			sub.TriggeredContainerIDs.Range(func(id string, _ struct{}) bool {
				containerIDs = append(containerIDs, id)
				return true
			})
		}

		var lastTriggered *time.Time
		if t := sub.LastTriggeredAt.Load(); t != nil && !t.IsZero() {
			lastTriggered = t
		}

		stats = append(stats, types.SubscriptionStats{
			SubscriptionID:        sub.ID,
			TriggerCount:          sub.TriggerCount.Load(),
			LastTriggeredAt:       lastTriggered,
			TriggeredContainerIDs: containerIDs,
		})
		return true
	})
	return stats
}

// Dispatchers returns all dispatchers as DispatcherConfig sorted by ID
func (m *Manager) Dispatchers() []DispatcherConfig {
	result := make([]DispatcherConfig, 0)
	m.dispatcherConfigs.Range(func(_ int, cfg DispatcherConfig) bool {
		result = append(result, cfg)
		return true
	})
	slices.SortFunc(result, func(a, b DispatcherConfig) int {
		return a.ID - b.ID
	})
	return result
}

func (m *Manager) Templates() []*NotificationTemplate {
	result := make([]*NotificationTemplate, 0)
	m.templates.Range(func(_ int, tmpl *NotificationTemplate) bool {
		copyTemplate := *tmpl
		result = append(result, &copyTemplate)
		return true
	})
	slices.SortFunc(result, func(a, b *NotificationTemplate) int { return a.ID - b.ID })
	return result
}

func (m *Manager) AddTemplate(tmpl *NotificationTemplate) *NotificationTemplate {
	tmpl.ID = int(m.templateCounter.Add(1))
	copyTemplate := *tmpl
	m.templates.Store(tmpl.ID, &copyTemplate)
	return &copyTemplate
}

func (m *Manager) UpdateTemplate(id int, tmpl *NotificationTemplate) (*NotificationTemplate, error) {
	tmpl.ID = id
	copyTemplate := *tmpl
	m.templates.Store(id, &copyTemplate)
	m.rebuildDispatchersForTemplate(id)
	return &copyTemplate, nil
}

func (m *Manager) DeleteTemplate(id int) {
	m.templates.Delete(id)
	m.rebuildDispatchersForTemplate(id)
}

func (m *Manager) ResolveTemplate(templateID int, inline string) string {
	return m.resolveTemplate(templateID, inline)
}

func (m *Manager) resolveTemplate(templateID int, inline string) string {
	if templateID > 0 {
		if tmpl, ok := m.templates.Load(templateID); ok {
			return tmpl.Body
		}
	}
	return inline
}

func (m *Manager) rebuildDispatchersForTemplate(templateID int) {
	m.dispatcherConfigs.Range(func(id int, cfg DispatcherConfig) bool {
		if cfg.TemplateID != templateID {
			return true
		}
		d, err := m.createDispatcher(cfg)
		if err != nil {
			log.Warn().Err(err).Int("dispatcher", id).Msg("Failed to rebuild dispatcher after template change")
			return true
		}
		m.dispatchers.Store(id, d)
		return true
	})
}

func (m *Manager) createDispatcher(config DispatcherConfig) (dispatcher.Dispatcher, error) {
	resolved := config
	resolved.Template = m.resolveTemplate(config.TemplateID, config.Template)
	return createDispatcher(resolved)
}
